// Execute the shipped script with a small DOM/API harness; no browser dependencies.
const {test} = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, '../frontend/assets/app.js'), 'utf8');

async function app() {
  const elements = new Map();
  function element(id) {
    if (!elements.has(id)) {
      let value = '';
      const classes = new Set();
      elements.set(id, {
        style: {}, dataset: {}, textContent: '', innerHTML: '', children: [],
        get value() { return value; }, set value(v) { value = String(v); },
        classList: {add: c=>classes.add(c), remove: c=>classes.delete(c), contains: c=>classes.has(c), toggle: ()=>{}},
        appendChild(e) { this.children.push(e); }, append(...e) { this.children.push(...e); },
        replaceChildren(...e) { this.children = e; }, add(e) { this.children.push(e); },
        remove() {}, setAttribute() {}, querySelector() { return null; },
      });
    }
    return elements.get(id);
  }
  const calls = [], notices = [];
  let reply = () => [];
  const context = vm.createContext({
    window: {addEventListener() {}, location: {origin:'http://localhost', search:''}},
    document: {getElementById: element, querySelector: element, querySelectorAll: ()=>[],
      createElement: ()=>element(Symbol()), addEventListener() {}, body:element('body'), documentElement:element('html')},
    localStorage: {getItem: ()=>null, setItem() {}, removeItem() {}},
    setInterval: ()=>1, clearInterval() {}, setTimeout: ()=>1, clearTimeout() {},
    URL, URLSearchParams, console, history: {replaceState() {}},
    Option: function(text, value) { this.textContent=text; this.value=value; },
    fetch: async()=>({ok:true,status:200,headers:{get:()=>'application/json'},json:async()=>[]}),
    request: async(method, url, body)=>{ calls.push({method,url,body}); return reply(method,url,body); },
    notice: msg=>notices.push(msg),
  });
  vm.runInContext(source, context);
  await new Promise(resolve=>setImmediate(resolve));
  const run = code=>vm.runInContext(code, context);
  run("api=request; showClientNotification=notice; toast=()=>{}; state.user={id:1,role:'client'}; state.token='test';");
  return {run, element, calls, notices, respond(fn) { reply=fn; }};
}

test('frozen, expired, upcoming and exhausted subscriptions are excluded from active selection', async()=>{
  const a=await app();
  a.run(`var now=new Date('2026-09-24T12:00:00Z'); var base={is_active:true,start_date:'2026-09-01',end_date:'2026-10-01',sessions_left:5};`);
  assert.equal(a.run('subscriptionState({...base,frozen_at:"2026-09-23"},now)'), 'frozen');
  assert.equal(a.run('subscriptionState({...base,end_date:now},now)'), 'expired');
  assert.equal(a.run('subscriptionState({...base,start_date:"2026-10-01"},now)'), 'upcoming');
  assert.equal(a.run('subscriptionState({...base,sessions_left:0},now)'), 'inactive');
  assert.equal(a.run('activeSubscription([{...base,id:3,end_date:"2026-11-01"},{...base,id:1,frozen_at:"2026-09-23"},{...base,id:2}],now).id'), 2);
  assert.equal(a.run('activeSubscription([{...base,sessions_left:null}],now).sessions_left'), null);
});

test('client day click and next month reload the visible calendar from the API', async()=>{
  const a=await app();
  await a.run('calClickDay(2026,8,24)');
  assert.match(a.element('client-calendar').innerHTML, /cal-day-view/);
  assert.equal(a.element('staff-calendar').innerHTML, '');
  assert.match(a.calls[0].url, /date_from=2026-09-23&date_to=2026-09-25/);
  a.run("state.calView='month'; state.calDate=new Date(2026,8,1);");
  await a.run("calNext('client-calendar')");
  assert.match(a.calls.at(-1).url, /date_from=2026-09-30&date_to=2026-11-01/);
});

test('late calendar responses cannot replace a newer selected period', async()=>{
  const a=await app();
  let release;
  a.respond(()=>new Promise(resolve=>{release=resolve;}));
  const earlier=a.run('loadClientSchedule()');
  a.respond(()=>[{id:2,start_time:'2026-10-10',status:'scheduled',title:'New'}]);
  await a.run("calNext('client-calendar')");
  release([{id:1}]); await earlier;
  assert.equal(a.run('state.trainings[0].id'), 2);
});

test('the first approved request notifies once, including when the initial count is zero', async()=>{
  const a=await app();
  let rows=[];
  a.respond((_,url)=>url==='/schedule/requests/my'?rows:[]);
  await a.run('pollClientRequests()');
  rows=[{id:7,status:'scheduled'}];
  await a.run('pollClientRequests()'); await a.run('pollClientRequests()');
  assert.equal(a.notices.length,1);
  a.run("clearAuth(); state.user={id:2,role:'client'}; state.token='second';");
  await a.run('pollClientRequests()');
  assert.equal(a.notices.length,1, 'existing approvals in a different account are only a baseline');
});

test('trainer card selects trainer mode and submits trainer_id', async()=>{
  const a=await app();
  a.respond((_,url)=>url==='/trainers'?[{id:9,full_name:'Тренер',specialization:'fitness'}]:[]);
  await a.run('openTrainingRequestModalWithTrainer(9)');
  assert.equal(a.run('_reqType'), 'trainer');
  assert.equal(a.element('req-trainer').value,'9');
  a.element('req-date').value='2099-10-10';
  a.element('#req-time-grid button.time-sel').dataset.time='12:00';
  await a.run('submitTrainingRequest()');
  const sent=a.calls.find(c=>c.method==='POST');
  assert.equal(sent.url,'/schedule/requests');
  assert.equal(sent.body.trainer_id,9);
});

test('approval preserves requested trainer and permits explicitly removing the trainer', async()=>{
  const a=await app();
  a.respond((_,url)=>url==='/schedule/requests'?[{id:7,client_id:1,trainer_id:9,preferred_at:'2099-10-10T23:30:00',client_name:'Клиент'}]:url==='/trainers'?[{id:9,full_name:'Тренер'}]:[]);
  await a.run('openApproveRequestModal(7)');
  assert.equal(a.element('ar-trainer').value,'9');
  assert.equal(a.element('ar-end').value,'11.10.2099 00:30');
  await a.run('submitApproveRequest()');
  let sent=a.calls.filter(c=>c.method==='POST').at(-1);
  assert.equal(sent.body.trainer_id,9); assert.equal(sent.body.clear_trainer,false);
  a.element('ar-trainer').value='';
  await a.run('submitApproveRequest()');
  sent=a.calls.filter(c=>c.method==='POST').at(-1);
  assert.equal(sent.body.trainer_id,null); assert.equal(sent.body.clear_trainer,true);
});

test('weekly statistics include completed visits from Monday before today', async()=>{
  const a=await app();
  a.run(`var RealDate=Date; Date=class extends RealDate { constructor(...args){super(...(args.length?args:['2026-09-24T12:00:00']));} static now(){return new RealDate('2026-09-24T12:00:00').getTime();} };`);
  a.respond((_,url)=>url.startsWith('/schedule?')?[{status:'completed',start_time:'2026-09-21T12:00:00'}]:[]);
  await a.run('loadClientStats()');
  assert.match(a.calls.find(c=>c.url.startsWith('/schedule?')).url,/date_from=2026-09-20/);
  assert.equal(a.element('stat-week').textContent,1);
});

test('product edit keeps the existing subscription type in the submitted payload', async()=>{
  const a=await app();
  a.run("openEditProductModal({id:5,name:'Разовый',price:100,category:'subscription',sub_type:'single',sessions_count:1,duration_days:1})");
  await a.run('submitCreateProduct()');
  const sent=a.calls.find(c=>c.method==='PUT');
  assert.equal(sent.body.sub_type,'single');
});


test('password submission is single flight and a failed link explains recovery', async()=>{
  const a=await app();
  a.run("accountAction={action:'activate',token:'pending'};");
  a.element('set-password-new').value='validpass123';a.element('set-password-confirm').value='validpass123';
  let reject;a.respond(()=>new Promise((_,r)=>{reject=r;}));
  const first=a.run('submitSetPassword()');await a.run('submitSetPassword()');
  assert.equal(a.calls.length,1);assert.equal(a.element('set-password-submit').disabled,true);
  const error=new Error('invalid');error.code='invalid_account_link';reject(error);await first;
  assert.match(a.element('set-password-msg').textContent,/попросите администратора/);
  assert.equal(a.element('set-password-submit').disabled,false);
  a.respond(()=>({}));await a.run('submitSetPassword()');await a.run('submitSetPassword()');
  assert.equal(a.calls.length,2);assert.match(a.element('set-password-msg').textContent,/Пароль сохранён/);
});

test('catalog shows safe product photos for either category and escapes text', async()=>{
  const a=await app();
  const photo='/uploads/products/'+'a'.repeat(24)+'.jpg';
  for(const category of ['subscription','sports']){
    a.run(`renderShopGrid([{id:1,name:'<bad>',category:'${category}',photo_url:'${photo}'}],'shop-grid',false)`);
    assert.match(a.element('shop-grid').innerHTML,/img src="\/uploads\/products\//);
    assert.match(a.element('shop-grid').innerHTML,/alt="&lt;bad&gt;"/);
  }
  assert.equal(a.run("safeImageURL('javascript:alert(1)')"),'');
  assert.equal(a.run("safeImageURL('/uploads/products/../secret')"),'');
});

test('retrying a photo failure edits the created product without duplicating it', async()=>{
  const a=await app();a.run('openCreateProductModal();_productPhotoBlob={};uploadProductPhoto=async()=>{throw new Error("offline")};');
  a.element('cp-name').value='Фото товар';a.element('cp-price').value='100';a.element('cp-cat').value='sports';
  a.respond((method)=>method==='POST'?{id:77}:[]);
  await a.run('submitCreateProduct()');
  assert.equal(a.run('state.editingProductId'),77);assert.match(a.element('cp-err').textContent,/Товар сохранён/);
  a.run('uploadProductPhoto=async()=>({});');await a.run('submitCreateProduct()');
  assert.equal(a.calls.filter(x=>x.method==='POST').length,1);
  assert.equal(a.calls.filter(x=>x.method==='PUT'&&x.url==='/shop/products/77').length,1);
});

test('removing an existing product photo is persisted separately from product fields', async()=>{
  const a=await app();a.run("openEditProductModal({id:9,name:'Товар',price:20,category:'sports',photo_url:'/uploads/products/aaaaaaaaaaaaaaaaaaaaaaaa.jpg'});removeProductPhotoSelection();");
  await a.run('submitCreateProduct()');
  assert.ok(a.calls.some(x=>x.method==='DELETE'&&x.url==='/shop/products/9/photo'));
});


test('activation URL copied from JSON logs retains the exact issued token', async()=>{
  const a=await app();
  a.run(`var issued='Ab_-'.repeat(10)+'xyz';`);
  assert.equal(a.run('issued.length'),43);
  a.run(`var line=JSON.stringify({body:'Откройте http://localhost/?action=activate&token='+issued+'\\n\\nСсылка действует сутки.'}); var copied=line.match(/http[^" ]+/)[0];`);
  // JSON body contains escaped line separators; mimic terminal URL selection.
  a.run(`window.location.search=new URL(copied).search;`);
  assert.equal(a.run('handleAccountActionFromURL()'),true);
  assert.equal(a.run('accountAction.token'),a.run('issued'));
  a.element('set-password-new').value='newpass123';a.element('set-password-confirm').value='newpass123';
  await a.run('submitSetPassword()');
  assert.equal(a.calls.at(-1).body.token,a.run('issued'));
});

test('URL cleanup preserves intact tokens and does not truncate arbitrary invalid suffixes', async()=>{
 const a=await app();a.run("var issued='A'.repeat(43)");
 assert.equal(a.run('accountTokenFromURL(issued)'),a.run('issued'));
 assert.equal(a.run("accountTokenFromURL(issued+'extra')"),a.run("issued+'extra'"));
 assert.equal(a.run("accountTokenFromURL('short')"),'short');
 assert.equal(a.run("accountTokenFromURL(issued+'%5Cn')"),a.run("issued+'%5Cn'"));
});
