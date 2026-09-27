const API = (window.SFEDU_API_BASE || '/api/v1').replace(/\/+$/, '');
let state = {
  user:null,token:null,trainings:[],trainers:[],clients:[],applications:[],
  products:[],payments:[],subscriptions:[],financeSummary:null,
  calView:'month',calDate:new Date(),
  currentPage:'landing',currentClientSection:'profile',currentStaffSection:'schedule',
  editingTrainingId:null,approvingAppId:null,editingProductId:null,editingClientId:null,
  editingTrainerId:null,shopFilter:'all',trainerFilter:'all',
  aiConversationId:null,aiSending:false,
};
let accountAction={action:'',token:''};
try{const saved=localStorage.getItem('sfedu_auth');if(saved){const d=JSON.parse(saved);state.user=d.user||null;state.token=d.token||null;}}catch{}

function esc(value=''){return String(value).replace(/[&<>"']/g,ch=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[ch]);}
function safeImageURL(value=''){const raw=String(value||'').trim();if(!raw)return'';if(/^\/uploads\/(?:trainers\/[a-f0-9]{24}\.(?:webp|jpg)|products\/[a-f0-9]{24}\.jpg)$/i.test(raw))return raw;try{const u=new URL(raw);return(u.protocol==='http:'||u.protocol==='https:')?u.href:'';}catch{return'';}}
function dateParam(d){const pad=n=>String(n).padStart(2,'0');return d.getFullYear()+'-'+pad(d.getMonth()+1)+'-'+pad(d.getDate());}
function saveAuth(){try{if(state.user&&state.token)localStorage.setItem('sfedu_auth',JSON.stringify({user:state.user,token:state.token}));}catch{}}
function clearAuth(){_clientRequestStates=null;_clientRequestUserID=null;_calendarLoadVersion++;state.user=null;state.token=null;state.aiConversationId=null;try{localStorage.removeItem('sfedu_auth');}catch{}}
function aiConversationStorageKey(){return state.user?.id?'sfedu_ai_conv_'+state.user.id:'';}
function restoreAIConversation(){state.aiConversationId=null;const key=aiConversationStorageKey();if(!key)return;try{const v=parseInt(localStorage.getItem(key),10);if(Number.isInteger(v)&&v>0)state.aiConversationId=v;}catch{}}
function persistAIConversation(){const key=aiConversationStorageKey();if(!key)return;try{if(state.aiConversationId)localStorage.setItem(key,String(state.aiConversationId));else localStorage.removeItem(key);}catch{}}

async function api(method,path,body){
  const headers={};if(state.token)headers.Authorization='Bearer '+state.token;if(body!==undefined&&body!==null)headers['Content-Type']='application/json';
  let r;try{r=await fetch(API+path,{method,headers,...(body!==undefined&&body!==null?{body:JSON.stringify(body)}:{})});}catch{throw new Error('Не удалось подключиться к серверу. Убедитесь, что Go-backend запущен.');}
  let data=null;if(r.status!==204){const ct=r.headers.get('content-type')||'';data=ct.includes('application/json')?await r.json().catch(()=>null):await r.text().catch(()=>'');}
  if(!r.ok){if(r.status===401&&state.token){clearAuth();showPage('page-landing');const ai=document.getElementById('ai-panel');if(ai)ai.style.display='none';}
    const msg=data&&typeof data==='object'?data.error:'';const mapped={400:'Некорректные данные',401:'Сессия истекла. Войдите снова',403:'Недостаточно прав',404:'Объект не найден',409:'Конфликт операции',429:'Слишком много запросов',503:'Сервис временно недоступен'}[r.status];const error=new Error(msg||mapped||('Ошибка сервера: HTTP '+r.status));error.code=data?.code;throw error;}
  return data;
}
const GET    = p    => api('GET',    p);
const POST   = (p,b)=> api('POST',   p, b);
const PUT    = (p,b)=> api('PUT',    p, b);
const PATCH  = p    => api('PATCH',  p);
const DELETE = p    => api('DELETE', p);

function initials(name=''){return name.split(' ').slice(0,2).map(w=>w[0]||'').join('').toUpperCase()||'??';}
function fmt(iso){if(!iso)return'—';const d=new Date(iso);return d.toLocaleString('ru',{day:'2-digit',month:'2-digit',year:'numeric',hour:'2-digit',minute:'2-digit'});}
function fmtDate(iso){if(!iso)return'—';return new Date(iso).toLocaleDateString('ru');}
function fmtMoney(n){return(parseFloat(n)||0).toFixed(2);}
function subscriptionState(sub, now=new Date()) {
  if(!sub.is_active || (sub.sessions_left!=null && sub.sessions_left<=0))return 'inactive';
  if(sub.frozen_at)return 'frozen';
  if(new Date(sub.end_date)<=now)return 'expired';
  if(new Date(sub.start_date)>now)return 'upcoming';
  return 'active';
}
function activeSubscription(subs,now=new Date()) {
  return (subs||[]).filter(s=>subscriptionState(s,now)==='active')
    .sort((a,b)=>new Date(a.end_date)-new Date(b.end_date)||a.id-b.id)[0];
}
function subscriptionBadge(sub){
  const status=subscriptionState(sub),labels={active:'Активен',frozen:'Заморожен',expired:'Истёк',upcoming:'Ещё не начался',inactive:'Неактивен'};
  const color={active:'green',frozen:'yellow',expired:'red',upcoming:'blue',inactive:'gray'}[status];
  return '<span class="badge badge-'+color+'">'+labels[status]+'</span>';
}
function statusLabel(s){return{scheduled:'Запланировано',completed:'Завершено',cancelled:'Отменено',pending:'Ожидает'}[s]||s;}
function statusBadge(s){const m={scheduled:'badge-blue',completed:'badge-green',cancelled:'badge-red',pending:'badge-yellow'};return '<span class="badge '+(m[s]||'badge-gray')+'">'+esc(statusLabel(s))+'</span>';}
function specLabel(s){return{body_relief:'Рельеф тела',weight_loss:'Похудение',mass_gain:'Набор массы'}[s]||s;}
function opLabel(s){return{income:'Доход',expense:'Расход',refund:'Возврат'}[s]||s;}
function roleLabel(r){return{client:'Клиент',manager:'Менеджер',admin:'Администратор'}[r]||r;}
function svcLabel(s){return{subscription:'Абонемент',training:'Тренировка',product:'Товар',deposit:'Пополнение'}[s]||s;}

function toast(msg,type='success'){
  const el=document.createElement('div'),isErr=type==='error';el.style.cssText='position:fixed;bottom:28px;right:28px;z-index:9999;background:var(--bg2);border:1px solid '+(isErr?'var(--red)':'var(--green)')+';border-radius:var(--r2);padding:14px 18px;font-size:14px;box-shadow:0 8px 32px rgba(0,0,0,.5);display:flex;align-items:center;gap:12px;max-width:360px';
  const icon=document.createElement('div');icon.textContent=isErr?'×':'✓';icon.style.cssText='font-weight:800;color:'+(isErr?'#f87171':'#4ade80');
  const text=document.createElement('span');text.textContent=String(msg||'');text.style.cssText='flex:1;color:var(--t1)';
  const close=document.createElement('button');close.textContent='×';close.style.cssText='background:none;border:none;color:var(--t3)';close.onclick=()=>el.remove();el.append(icon,text,close);document.body.appendChild(el);setTimeout(()=>el.remove(),3800);
}
function showErr(id,msg){const el=document.getElementById(id);if(el){el.textContent=msg;el.style.display='block';}}
function hideErr(id){const el=document.getElementById(id);if(el)el.style.display='none';}

function openModal(id){document.getElementById(id).classList.add('open');}
function closeModal(id){document.getElementById(id).classList.remove('open');}
document.addEventListener('click',e=>{if(e.target.classList.contains('modal-overlay'))e.target.classList.remove('open');});

function toggleDropdown(id){
  const el=document.getElementById(id);
  const isOpen=el.style.display!=='none';
  closeAllDropdowns();
  if(!isOpen)el.style.display='block';
}
function closeAllDropdowns(){document.querySelectorAll('.dropdown-menu').forEach(m=>m.style.display='none');}
document.addEventListener('click',e=>{if(!e.target.closest('.dropdown'))closeAllDropdowns();});

function toggleClientMenu(id){
  const menu=document.getElementById('cmenu-'+id);
  if(!menu)return;
  const isOpen=menu.style.display!=='none';
  closeAllClientMenus();
  if(!isOpen)menu.style.display='block';
}
function closeAllClientMenus(){document.querySelectorAll('[id^="cmenu-"]').forEach(m=>m.style.display='none');}
document.addEventListener('click',e=>{
  if(!e.target.closest('[id^="cmenu-"]')&&!e.target.closest('button[onclick*="toggleClientMenu"]'))closeAllClientMenus();
});

function showPage(id){
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.getElementById(id).classList.add('active');
  state.currentPage=id;
  if(id==='page-trainers')loadTrainersPublic();
  if(id==='page-landing')renderMiniCal();
}
function scrollToSection(id){
  showPage('page-landing');
  setTimeout(()=>{const el=document.getElementById(id);if(el)el.scrollIntoView({behavior:'smooth'});},100);
}

function parseRuDate(str){
  if(!str)return'';
  str=str.trim();
  if(str.includes('T')||(str.includes('-')&&!str.includes('.'))){return new Date(str).toISOString();}
  const parts=str.split(' ');
  const datePart=parts[0]||'';
  const timePart=parts[1]||'00:00';
  const dp=datePart.split('.');
  const tp=timePart.split(':');
  const d=parseInt(dp[0])||1,m=parseInt(dp[1])||1,y=parseInt(dp[2])||2026;
  const h=parseInt(tp[0])||0,min=parseInt(tp[1])||0;
  return new Date(y,m-1,d,h,min).toISOString();
}

// AUTH
async function submitLogin(){
  const email=document.getElementById('l-email').value.trim(),password=document.getElementById('l-pass').value;hideErr('login-err');if(!email){showErr('login-err','Введите email');return;}if(!password){showErr('login-err','Введите пароль');return;}
  try{const data=await POST('/auth/login',{email,password});if(!data?.user||!data?.token)throw new Error('Некорректный ответ авторизации');state.user=data.user;state.token=data.token;saveAuth();restoreAIConversation();closeModal('modal-login');enterDashboard();}catch(e){showErr('login-err',e.message||'Неверный email или пароль');}
}

function logout(){if(typeof _pollingInterval!=='undefined'&&_pollingInterval)clearInterval(_pollingInterval);if(typeof _clientPollingInterval!=='undefined'&&_clientPollingInterval)clearInterval(_clientPollingInterval);clearAuth();const ai=document.getElementById('ai-panel');if(ai)ai.style.display='none';resetAIMessages();showPage('page-landing');}

function enterDashboard(){
  if(!state.user)return;
  if(state.user.role==='client'){
    showPage('page-client');populateClientUI();clientNav('profile');startClientPolling();
  }else{
    showPage('page-staff');populateStaffUI();staffNav('schedule');
  }
}

window.addEventListener('load',async()=>{renderMiniCal();if(handleAccountActionFromURL())return;if(state.token){try{state.user=await GET('/users/me');saveAuth();restoreAIConversation();enterDashboard();}catch{clearAuth();showPage('page-landing');}}});

// A URL copied from a JSON email log can include literal \\n escapes and body text.
// Recover only the complete 32-byte base64url token before that line separator.
function accountTokenFromURL(value){
  const raw=String(value||'').trim(),match=raw.match(/^([A-Za-z0-9_-]{43})(?=\\[nr]|[\r\n]|$)/);
  return match?match[1]:raw;
}
function handleAccountActionFromURL(){
  const params=new URLSearchParams(window.location.search),action=params.get('action')||'',token=accountTokenFromURL(params.get('token'));
  if(!token||!['activate','reset'].includes(action))return false;
  accountAction={action,token};history.replaceState({},document.title,window.location.pathname);clearAuth();showPage('page-landing');
  document.getElementById('set-password-title').textContent=action==='activate'?'Создайте пароль для входа':'Создайте новый пароль';
  document.getElementById('set-password-hint').textContent=action==='activate'?'Ваша заявка одобрена. Придумайте пароль, который будете использовать для входа в SFEDU.':'Придумайте новый пароль для аккаунта SFEDU.';
  document.getElementById('set-password-new').value='';document.getElementById('set-password-confirm').value='';document.getElementById('set-password-submit').disabled=false;hideErr('set-password-msg');openModal('modal-set-password');return true;
}

function openForgotPassword(){const el=document.getElementById('forgot-email');if(el)el.value=document.getElementById('l-email')?.value||'';hideErr('forgot-msg');openModal('modal-forgot-password');}

async function submitForgotPassword(){
  const email=document.getElementById('forgot-email').value.trim(),msg=document.getElementById('forgot-msg');msg.style.display='none';
  if(!email){msg.textContent='Введите email';msg.style.color='var(--red)';msg.style.display='block';return;}
  try{await POST('/auth/forgot-password',{email});msg.textContent='Если аккаунт с таким email существует, ссылка для восстановления отправлена.';msg.style.color='var(--green)';msg.style.display='block';}
  catch(e){msg.textContent=e.message||'Не удалось отправить запрос';msg.style.color='var(--red)';msg.style.display='block';}
}

let _passwordSubmitting=false;
async function submitSetPassword(){
  if(_passwordSubmitting||document.getElementById('set-password-submit').disabled)return;
  const password=document.getElementById('set-password-new').value,confirm=document.getElementById('set-password-confirm').value,msg=document.getElementById('set-password-msg'),button=document.getElementById('set-password-submit');msg.style.display='none';msg.style.color='var(--red)';
  if(!accountAction.token){showErr('set-password-msg','Откройте ссылку из приглашения заново.');return;}
  if(password.length<8){showErr('set-password-msg','Пароль должен содержать минимум 8 символов');return;}
  if(password!==confirm){showErr('set-password-msg','Пароли не совпадают');return;}
  const action=accountAction.action,endpoint=action==='activate'?'/auth/activate':'/auth/reset-password';
  _passwordSubmitting=true;button.disabled=true;
  try{
    await POST(endpoint,{token:accountAction.token,new_password:password});
    accountAction={action:'',token:''};history.replaceState({},document.title,window.location.pathname);
    msg.textContent='Пароль сохранён. Теперь можно войти в систему.';msg.style.color='var(--green)';msg.style.display='block';
    setTimeout(()=>{closeModal('modal-set-password');openModal('modal-login');},900);
  }catch(e){
    const invalid=e.code==='invalid_account_link'||/invalid or expired link/i.test(e.message||'');
    msg.textContent=invalid?'Ссылка недействительна, уже использована или истекла. '+(action==='activate'?'Если пароль уже создан, войдите в аккаунт. Иначе попросите администратора повторить приглашение и откройте новую ссылку полностью.':'Запросите новую ссылку через «Забыли пароль?» на экране входа.'):e.message||'Не удалось сохранить пароль. Попробуйте ещё раз.';
    msg.style.color='var(--red)';msg.style.display='block';button.disabled=false;
  }finally{_passwordSubmitting=false;}
}


// APPLY
async function submitApply(){
  const full_name=document.getElementById('a-name').value.trim();
  const phone=document.getElementById('a-phone').value.trim();
  const email=document.getElementById('a-email').value.trim();
  const gender=document.getElementById('a-gender').value;
  hideErr('apply-err');
  if(!gender){showErr('apply-err','Укажите пол');return;}
  if(!full_name||full_name.split(' ').length<2){showErr('apply-err','Введите полное ФИО');return;}
  if(!/^(\+7|8)[\s\-]?\(?\d{3}\)?[\s\-]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}$/.test(phone)){showErr('apply-err','Неверный формат телефона. Пример: +7(987)654-32-10');return;}
  if(!/^[a-zA-Z0-9._%+\-]+@(gmail\.com|mail\.ru)$/i.test(email)){showErr('apply-err','Принимаются только @gmail.com и @mail.ru');return;}
  try{
    await POST('/auth/apply',{full_name,phone,email,gender});
    document.getElementById('apply-form').style.display='none';
    document.getElementById('apply-success').style.display='block';
  }catch(e){showErr('apply-err',e.message||'Ошибка');}
}

function populateClientUI(){
  const u=state.user;if(!u)return;
  const ini=initials(u.full_name);
  document.getElementById('c-avatar').textContent=ini;
  document.getElementById('c-firstname').textContent=u.full_name.split(' ')[0]||'';
  document.getElementById('c-balance').textContent=fmtMoney(u.balance);
  document.getElementById('profile-big-avatar').textContent=ini;
  document.getElementById('profile-fullname').textContent=u.full_name;
  document.getElementById('p-fullname').value=u.full_name;
  document.getElementById('p-phone').value=u.phone;
  document.getElementById('p-email').value=u.email;
  const genderEl=document.querySelector('input[name="gender"][value="'+(u.gender||'male')+'"]');
  if(genderEl)genderEl.checked=true;
  ['p-fullname','p-phone','p-email'].forEach(id=>{
    const el=document.getElementById(id);
    if(el){el.readOnly=true;el.style.opacity='0.6';el.style.cursor='default';}
  });
  document.getElementById('gender-male').disabled=true;
  document.getElementById('gender-female').disabled=true;
  document.getElementById('btn-save-profile').style.display='none';
  document.getElementById('btn-edit-profile').style.display='inline-flex';
  loadClientStats();
}

async function loadClientStats() {
  // Скелетон пока грузится
  document.getElementById('stat-visits').textContent = '—';
  document.getElementById('stat-sessions').textContent = '—';
  document.getElementById('stat-sub-expires').textContent = '—';
  const nextEl = document.getElementById('stat-next-training');
  nextEl.textContent = 'Загрузка...';
  nextEl.style.color = 'var(--t2)';
  const weekEl = document.getElementById('stat-week');
  if(weekEl) weekEl.textContent = '—';

  try {
    document.getElementById('stat-visits').textContent = state.user?.visits || 0;

    const subs = await GET('/shop/my-subscriptions');
    const activeSub = activeSubscription(subs);
    if(activeSub) {
      document.getElementById('stat-sessions').textContent = activeSub.sessions_left ?? '∞';
      const exp = new Date(activeSub.end_date);
      const daysLeft = Math.ceil((exp - new Date()) / 864e5);
      const el = document.getElementById('stat-sub-expires');
      el.textContent = fmtDate(activeSub.end_date);
      el.style.color = daysLeft <= 3 ? 'var(--red)' : daysLeft <= 7 ? 'var(--yellow)' : 'var(--green)';
      if(daysLeft <= 3 && daysLeft > 0) {
        toast('Абонемент истекает через '+daysLeft+' дн. Продлите его в Магазине.', 'error');
      }
    } else {
      document.getElementById('stat-sessions').textContent = '0';
      const el = document.getElementById('stat-sub-expires');
      el.textContent = 'Нет'; el.style.color = 'var(--t3)';
    }

    const now = new Date();
    const weekStart = new Date(now); weekStart.setDate(now.getDate()-(now.getDay()||7)+1); weekStart.setHours(0,0,0,0);
    const from = dateParam(new Date(weekStart.getTime()-864e5));
    const to = dateParam(new Date(now.getFullYear(), now.getMonth()+2, 0));
    const trainings = await GET('/schedule?date_from='+from+'&date_to='+to);

    const upcoming = (trainings||[])
      .filter(t => t.status==='scheduled' && new Date(t.start_time) > now)
      .sort((a,b) => new Date(a.start_time) - new Date(b.start_time));
    if(upcoming.length) {
      nextEl.textContent = upcoming[0].title + ' — ' + fmt(upcoming[0].start_time);
      nextEl.style.color = 'var(--t1)';
    } else {
      nextEl.textContent = 'Нет запланированных тренировок';
      nextEl.style.color = 'var(--t2)';
    }

    const dow2 = now.getDay()===0?6:now.getDay()-1;
    const mon = new Date(now); mon.setDate(now.getDate()-dow2); mon.setHours(0,0,0,0);
    const sun = new Date(mon); sun.setDate(mon.getDate()+6); sun.setHours(23,59,59,999);
    const weekCount = (trainings||[]).filter(t => {
      const d = new Date(t.start_time);
      return d >= mon && d <= sun && t.status === 'completed';
    }).length;
    if(weekEl) weekEl.textContent = weekCount;

  } catch {}
}

function enableProfileEdit(){
  ['p-fullname','p-phone','p-email'].forEach(id=>{
    const el=document.getElementById(id);
    if(el){el.readOnly=false;el.style.opacity='1';el.style.cursor='text';}
  });
  document.getElementById('gender-male').disabled=false;
  document.getElementById('gender-female').disabled=false;
  document.getElementById('btn-save-profile').style.display='inline-flex';
  document.getElementById('btn-edit-profile').style.display='none';
}

async function refreshMe(){
  try{const u=await GET('/users/me');state.user=u;saveAuth();populateClientUI();}catch{}
}

let _ctrClients = [];

function filterClientDropdown(val){const dropdown=document.getElementById('ctr-client-dropdown');if(!val.trim()){dropdown.style.display='none';return;}const filtered=_ctrClients.filter(u=>String(u.full_name||'').toLowerCase().includes(val.toLowerCase()));if(!filtered.length){dropdown.style.display='none';return;}dropdown.style.display='block';dropdown.innerHTML=filtered.map(u=>'<div onclick="selectClientDropdown('+Number(u.id)+')" style="padding:10px 14px;cursor:pointer;font-size:14px;border-bottom:1px solid var(--border)"><div style="font-weight:600">'+esc(u.full_name)+'</div><div style="font-size:12px;color:var(--t2)">'+esc(u.phone)+' · '+esc(u.email)+'</div></div>').join('');}

function selectClientDropdown(id){const u=_ctrClients.find(x=>x.id===id);if(!u)return;document.getElementById('ctr-client').value=id;document.getElementById('ctr-client-search').value=u.full_name;document.getElementById('ctr-client-dropdown').style.display='none';}

let _reqType='solo';

function setReqType(type){
  _reqType=type;
  document.getElementById('req-type-solo').className=type==='solo'?'btn btn-primary btn-sm':'btn btn-ghost btn-sm';
  document.getElementById('req-type-trainer').className=type==='trainer'?'btn btn-primary btn-sm':'btn btn-ghost btn-sm';
  document.getElementById('req-date-block').style.display=type==='trainer'?'block':'none';
  document.getElementById('req-trainer-time-block').style.display=type==='trainer'?'block':'none';
}

function clientNav(sec){
  ['profile','schedule','progress','notifications','payments','shop','trainers'].forEach(s=>{
    document.getElementById('csec-'+s).style.display='none';
    document.getElementById('cnav-'+s).classList.remove('active');
  });
  const csec=document.getElementById('csec-'+sec);
  csec.style.display=sec==='schedule'?'flex':'block';
  csec.style.animation='none';
  csec.offsetHeight;
  csec.style.animation='fadeUp .25s ease';
  document.getElementById('cnav-'+sec).classList.add('active');
  state.currentClientSection=sec;
  if(sec==='schedule'){loadClientSchedule();startClientPolling();}
  if(sec==='payments')loadClientPayments();
  if(sec==='shop')loadShop();
  if(sec==='trainers')loadClientTrainers();
  if(sec==='progress')loadProgress();
  if(sec==='notifications')loadNotifications();
}

let _clientPollingInterval = null;
let _clientRequestStates=null;
let _clientRequestUserID=null;
let _clientPollingBusy=false;

function startClientPolling() {
  if(_clientPollingInterval)clearInterval(_clientPollingInterval);
  if(_clientRequestUserID!==state.user?.id){_clientRequestStates=null;_clientRequestUserID=state.user?.id;}
  pollClientRequests();
  _clientPollingInterval=setInterval(pollClientRequests,10000);
}
async function pollClientRequests(){
  if(_clientPollingBusy||!state.user||state.user.role!=='client')return;
  const userID=state.user.id,token=state.token;_clientPollingBusy=true;
  try{
    const rows=await GET('/schedule/requests/my');
    if(userID!==state.user?.id||token!==state.token)return;
    const next=new Map((rows||[]).map(r=>[r.id,r.status]));
    const approved=(rows||[]).some(r=>r.status==='scheduled'&&_clientRequestStates!==null&&_clientRequestStates.get(r.id)!=='scheduled');
    _clientRequestStates=next;
    if(approved){showClientNotification('Ваша заявка на тренировку принята! Проверьте расписание.');await loadClientSchedule();}
  }catch{}finally{_clientPollingBusy=false;}
}

function showClientNotification(msg){
 const el=document.createElement('div');el.style.cssText='position:fixed;top:24px;right:24px;z-index:9999;background:var(--bg2);border:1px solid var(--green);border-radius:var(--r2);padding:16px 20px;max-width:320px;box-shadow:0 8px 32px rgba(0,0,0,.5);display:flex;align-items:flex-start;gap:12px';
 const icon=document.createElement('div');icon.textContent='✓';icon.style.cssText='font-size:20px;color:var(--green)';const box=document.createElement('div');box.style.flex='1';const title=document.createElement('div');title.textContent='Заявка принята';title.style.cssText='font-size:14px;font-weight:700;margin-bottom:4px';const text=document.createElement('div');text.textContent=String(msg||'');text.style.cssText='font-size:13px;color:var(--t2);line-height:1.5';box.append(title,text);const close=document.createElement('button');close.textContent='×';close.style.cssText='background:none;border:none;color:var(--t3)';close.onclick=()=>el.remove();el.append(icon,box,close);document.body.appendChild(el);setTimeout(()=>el.remove(),6000);
 if('Notification'in window&&Notification.permission==='granted')new Notification('SFEDU',{body:String(msg||'')});
}

async function saveProfile(){
  const full_name=document.getElementById('p-fullname').value.trim();
  const phone=document.getElementById('p-phone').value.trim();
  const email=document.getElementById('p-email').value.trim();
  const gender=document.querySelector('input[name="gender"]:checked')?.value||'male';
  const msg=document.getElementById('profile-msg');msg.style.display='none';
  if(!full_name||full_name.split(' ').length<2){msg.textContent='Введите полное ФИО';msg.style.color='var(--red)';msg.style.display='block';return;}
  if(!/^(\+7|8)[\s\-]?\(?\d{3}\)?[\s\-]?\d{3}[\s\-]?\d{2}[\s\-]?\d{2}$/.test(phone)){msg.textContent='Неверный формат телефона';msg.style.color='var(--red)';msg.style.display='block';return;}
  if(!/^[a-zA-Z0-9._%+\-]+@(gmail\.com|mail\.ru)$/i.test(email)){msg.textContent='Принимаются только @gmail.com и @mail.ru';msg.style.color='var(--red)';msg.style.display='block';return;}
  try{
    const u=await PUT('/users/me',{full_name,phone,email,gender});
    state.user=u;saveAuth();
    populateClientUI();toast('Данные сохранены');
  }catch(e){toast(e.message||'Ошибка','error');}
}

function showPassMode(){document.getElementById('profile-view-mode').style.display='none';document.getElementById('profile-pass-mode').style.display='block';}
function hidePassMode(){document.getElementById('profile-pass-mode').style.display='none';document.getElementById('profile-view-mode').style.display='block';}

async function changePassword(){
  const old_password=document.getElementById('old-pass').value,new_password=document.getElementById('new-pass').value,confirm=document.getElementById('confirm-pass').value,msg=document.getElementById('pass-msg');msg.style.display='none';
  if(new_password!==confirm){msg.textContent='Пароли не совпадают';msg.style.color='var(--red)';msg.style.display='block';return;}if(new_password.length<8){msg.textContent='Новый пароль должен быть не короче 8 символов';msg.style.color='var(--red)';msg.style.display='block';return;}
  try{await PUT('/users/me/password',{old_password,new_password});toast('Пароль изменён. Войдите заново.');logout();setTimeout(()=>openModal('modal-login'),150);}catch(e){msg.textContent=e.message;msg.style.color='var(--red)';msg.style.display='block';}
}

function payTab(btn,tab){
  document.querySelectorAll('.pay-tab').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  ['history','invoices','subs'].forEach(t=>{
    const el=document.getElementById('pay-'+t+'-content');
    if(el)el.style.display=t===tab?'block':'none';
  });
}

async function loadClientPayments(){
  await refreshMe();
  const u=state.user;
  document.getElementById('pay-visits').textContent=u?.visits||0;
  document.getElementById('pay-balance').textContent=fmtMoney(u?.balance);
  try{
    const payments=await GET('/finance/me/payments');
    renderPaymentsTable(payments,'payments-tbody');
    const subs=await GET('/shop/my-subscriptions');
    renderSubsTable(subs);
    const activeSub=activeSubscription(subs);
    document.getElementById('pay-sessions').textContent=activeSub?(activeSub.sessions_left??'∞'):0;
  }catch{}
}

function renderPaymentsTable(list,tbodyId){
  const tb=document.getElementById(tbodyId);if(!list||!list.length){tb.innerHTML='<tr><td colspan="4" style="text-align:center;color:var(--t3);padding:24px">Нет данных</td></tr>';return;}
  tb.innerHTML=list.map(p=>'<tr><td>'+esc(fmtDate(p.created_at))+'</td><td>'+esc(p.client_name||state.user?.full_name||'—')+'</td><td style="font-weight:700;color:'+(p.operation_type==='income'?'var(--green)':'var(--red)')+'">'+esc(fmtMoney(p.amount))+' ₽</td><td>'+(p.operation_type==='income'?'<span class="badge badge-green">Пополнение</span>':p.operation_type==='refund'?'<span class="badge badge-yellow">Возврат</span>':'<span class="badge badge-red">Списание</span>')+'</td></tr>').join('');
}

function renderSubsTable(list){const tb=document.getElementById('subs-tbody');if(!list||!list.length){tb.innerHTML='<tr><td colspan="5" style="text-align:center;color:var(--t3);padding:24px">Абонементов нет</td></tr>';return;}tb.innerHTML=list.map(x=>'<tr><td>'+esc(x.product_name||'—')+'</td><td>'+esc(fmtDate(x.start_date))+'</td><td>'+esc(fmtDate(x.end_date))+'</td><td>'+esc(x.sessions_left!=null?x.sessions_left:'∞')+'</td><td>'+subscriptionBadge(x)+'</td></tr>').join('');}

async function loadShop(){
  const grid=document.getElementById('shop-grid');
  if(grid) grid.innerHTML=Array(4).fill('<div class="card" style="padding:20px;display:flex;flex-direction:column;gap:10px"><div class="skeleton" style="height:32px;width:32px;border-radius:8px"></div><div class="skeleton" style="height:16px;width:80%"></div><div class="skeleton" style="height:12px"></div><div class="skeleton" style="height:12px;width:60%"></div><div class="skeleton" style="height:28px;width:50%;margin-top:8px"></div><div class="skeleton" style="height:36px;margin-top:4px;border-radius:var(--r)"></div></div>').join('');
  try{const products=await GET('/shop/products');state.products=products||[];renderShopGrid(state.products,'shop-grid',false);}catch{}
}

function shopFilter(cat,btn){
  document.querySelectorAll('#shop-cat-filters .pay-tab').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  state.shopFilter=cat;
  const filtered=cat==='all'?state.products:state.products.filter(p=>p.category===cat);
  renderShopGrid(filtered,'shop-grid',false);
}

function renderShopGrid(list,gridId,isStaff){
  const grid=document.getElementById(gridId);if(!grid)return;if(!list||!list.length){grid.innerHTML='<p style="color:var(--t2);padding:20px">Товаров нет</p>';return;}const isAdmin=state.user?.role==='admin';
  grid.innerHTML=list.map(p=>{const icon=p.category==='subscription'?'<svg width="32" height="32" viewBox="0 0 32 32" fill="none"><rect width="32" height="32" rx="8" fill="rgba(37,99,235,0.2)"/><path d="M6 12h20M6 16h20M10 20h12" stroke="#3b82f6" stroke-width="2"/></svg>':'<svg width="32" height="32" viewBox="0 0 32 32" fill="none"><rect width="32" height="32" rx="8" fill="rgba(34,197,94,0.15)"/><path d="M10 22V14m6 8V10m6 12V17" stroke="#4ade80" stroke-width="2.5"/></svg>';
    let btns='';if(isStaff){btns='<div style="display:flex;flex-wrap:wrap;gap:8px;margin-top:14px"><button onclick="openEditProductModal_v2('+Number(p.id)+')" class="btn btn-ghost btn-sm" style="flex:1 1 120px">Изменить</button>'+(isAdmin?'<button onclick="deleteProduct('+Number(p.id)+')" class="btn btn-danger btn-sm" style="flex:1 1 120px">Деактивировать</button>':'')+'</div>';}else{btns='<button onclick="buyProduct('+Number(p.id)+')" class="btn btn-primary" style="width:100%;margin-top:14px">Купить</button>';}
    const photo=safeImageURL(p.photo_url);return '<div class="shop-card" style="display:flex;flex-direction:column"><div class="shop-product-media">'+(photo?'<img src="'+esc(photo)+'" alt="'+esc(p.name)+'" loading="lazy">':icon)+'</div><div style="font-size:15px;font-weight:700;margin-bottom:6px">'+esc(p.name)+'</div><div style="font-size:13px;color:var(--t2);margin-bottom:12px;line-height:1.5;flex:1">'+esc(p.description||'')+'</div>'+(p.sessions_count?'<div style="font-size:12px;color:var(--t3);margin-bottom:8px">'+esc(p.sessions_count)+' занятий · '+esc(p.duration_days)+' дней</div>':'')+'<div style="font-size:24px;font-weight:800;color:var(--blue2);font-family:var(--f2)">'+esc(fmtMoney(p.price))+' ₽</div>'+btns+'</div>';}).join('');
}

function openEditProductModal_v2(id){const p=state.products.find(x=>x.id===id);if(p)openEditProductModal(p);}

async function buyProduct(id){
  try{await POST('/shop/purchase',{product_id:id});toast('Куплено успешно!');await refreshMe();loadClientPayments();}
  catch(e){toast(e.message,'error');}
}

async function submitTopUp(){closeModal('modal-topup');toast('Пополнение выполняет менеджер или администратор.','error');}

async function loadClientTrainers(){
  try{const data=await GET('/trainers');state.trainers=data||[];renderClientTrainers(state.trainers);}catch{}
}

function clientTrainerFilter(spec,btn){
  document.querySelectorAll('#client-trainer-filters .pay-tab').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  const filtered=spec==='all'?state.trainers:state.trainers.filter(t=>t.specialization===spec);
  renderClientTrainers(filtered);
}

function renderClientTrainers(list){
 const grid=document.getElementById('client-trainers-grid');if(!list||!list.length){grid.innerHTML='<p style="color:var(--t2)">Тренеров нет</p>';return;}
 grid.innerHTML=list.map(t=>{const photo=safeImageURL(t.photo_url);return '<div class="trainer-card"><div class="trainer-photo">'+(photo?'<img src="'+esc(photo)+'" style="width:100%;height:100%;object-fit:cover" alt="">':'<span style="font-size:60px">💪</span>')+'</div><div class="trainer-info"><div class="trainer-name">'+esc(t.full_name)+'</div><div class="trainer-spec">'+esc(specLabel(t.specialization))+'</div><div class="trainer-bio">'+esc(t.bio||'')+'</div><div style="font-size:12px;color:var(--t3);margin-top:8px">Опыт: '+esc(t.experience_years)+' лет</div><button class="btn btn-primary btn-sm" style="margin-top:12px;width:100%" onclick="openTrainingRequestModalWithTrainer('+Number(t.id)+')">Записаться</button></div></div>';}).join('');
}

// CALENDAR
const MONTHS=['Январь','Февраль','Март','Апрель','Май','Июнь','Июль','Август','Сентябрь','Октябрь','Ноябрь','Декабрь'];
const DAYS=['ПН','ВТ','СР','ЧТ','ПТ','СБ','ВС'];
const DAY_NAMES=['Воскресенье','Понедельник','Вторник','Среда','Четверг','Пятница','Суббота'];

function calHeader(cid){
  const {calDate:d,calView:v}=state;
  const title=v==='day'?d.getDate()+' '+MONTHS[d.getMonth()]+' '+d.getFullYear():MONTHS[d.getMonth()]+' '+d.getFullYear();
  return '<div class="cal-nav">'
    +'<button class="cal-nav-btn" onclick="calPrev(\''+cid+'\')">‹</button>'
    +'<span class="cal-title">'+title+'</span>'
    +'<button class="cal-nav-btn" onclick="calNext(\''+cid+'\')">›</button>'
    +'<div class="cal-tabs">'
    +'<button class="cal-tab '+(v==='month'?'active':'')+'" onclick="setCalView(\'month\',\''+cid+'\')">Месяц</button>'
    +'<button class="cal-tab '+(v==='week'?'active':'')+'" onclick="setCalView(\'week\',\''+cid+'\')">Неделя</button>'
    +'<button class="cal-tab '+(v==='day'?'active':'')+'" onclick="setCalView(\'day\',\''+cid+'\')">День</button>'
    +'</div></div>';
}

function setCalView(v,cid){state.calView=v;return reloadCalendar(cid);}
function calPrev(cid){
  const d=state.calDate;
  if(state.calView==='month')state.calDate=new Date(d.getFullYear(),d.getMonth()-1,1);
  else if(state.calView==='week')state.calDate=new Date(d-7*864e5);
  else state.calDate=new Date(d-864e5);
  return reloadCalendar(cid);
}
function calNext(cid){
  const d=state.calDate;
  if(state.calView==='month')state.calDate=new Date(d.getFullYear(),d.getMonth()+1,1);
  else if(state.calView==='week')state.calDate=new Date(d.getTime()+7*864e5);
  else state.calDate=new Date(d.getTime()+864e5);
  return reloadCalendar(cid);
}

function getTrainingsForDay(date) {
  return state.trainings.filter(t => {
    const td = new Date(t.start_time);
    const matchDay = td.getFullYear()===date.getFullYear() && td.getMonth()===date.getMonth() && td.getDate()===date.getDate();
    const matchStatus = _schedStatusFilter === 'all' || t.status === _schedStatusFilter;
    return matchDay && matchStatus;
  });
}

function renderCalendar(cid){
  const wrap=document.getElementById(cid);if(!wrap)return;
  let html='<div class="cal-wrap">'+calHeader(cid);
  if(state.calView==='month')html+=renderMonthView();
  else if(state.calView==='week')html+=renderWeekView();
  else html+=renderDayView();
  html+='</div>';
  wrap.innerHTML=html;
}

function renderMonthView(){
  const d=state.calDate,firstDay=new Date(d.getFullYear(),d.getMonth(),1);let startDow=firstDay.getDay()-1;if(startDow<0)startDow=6;const daysInMonth=new Date(d.getFullYear(),d.getMonth()+1,0).getDate(),prevDays=new Date(d.getFullYear(),d.getMonth(),0).getDate(),today=new Date();let cells='';
  for(let i=startDow-1;i>=0;i--)cells+='<div class="cal-cell other-month"><span class="cal-date">'+(prevDays-i)+'</span></div>';
  for(let day=1;day<=daysInMonth;day++){const date=new Date(d.getFullYear(),d.getMonth(),day),isToday=date.toDateString()===today.toDateString(),ts=getTrainingsForDay(date);cells+='<div class="cal-cell'+(isToday?' today':'')+'" onclick="calClickDay('+d.getFullYear()+','+d.getMonth()+','+day+')"><span class="cal-date">'+day+'</span>'+ts.slice(0,2).map(t=>'<div class="cal-event '+esc(t.status)+'" onclick="event.stopPropagation();showTrainingDetail('+Number(t.id)+')">'+esc(new Date(t.start_time).toLocaleTimeString('ru',{hour:'2-digit',minute:'2-digit'}))+' '+esc(t.title)+'</div>').join('')+(ts.length>2?'<div style="font-size:11px;color:var(--t3);margin-top:2px">+'+(ts.length-2)+'</div>':'')+'</div>';}
  const total=startDow+daysInMonth,remaining=total%7===0?0:7-(total%7);for(let i=1;i<=remaining;i++)cells+='<div class="cal-cell other-month"><span class="cal-date">'+i+'</span></div>';
  return '<div class="cal-grid"><div class="cal-days-header">'+DAYS.map((x,i)=>'<div class="cal-day-hdr'+(i>=5?' weekend':'')+'">'+esc(x)+'</div>').join('')+'</div><div class="cal-cells">'+cells+'</div></div>';
}

function renderWeekView(){
 const d=state.calDate,dow=d.getDay()===0?6:d.getDay()-1,monday=new Date(d);monday.setDate(d.getDate()-dow);const today=new Date();
 const cols=DAYS.map((dayName,i)=>{const date=new Date(monday);date.setDate(monday.getDate()+i);const isToday=date.toDateString()===today.toDateString(),ts=getTrainingsForDay(date);return '<div class="cal-week-col"><div class="cal-week-hdr'+(isToday?' today':'')+'">'+esc(dayName)+' <span style="opacity:.6">'+date.getDate()+'</span></div><div class="cal-week-body">'+ts.map(t=>'<div class="cal-event '+esc(t.status)+'" style="padding:5px 8px;cursor:pointer" onclick="showTrainingDetail('+Number(t.id)+')"><div style="font-weight:700;font-size:12px">'+esc(new Date(t.start_time).toLocaleTimeString('ru',{hour:'2-digit',minute:'2-digit'}))+'</div><div>'+esc(t.title)+'</div></div>').join('')+'</div></div>';}).join('');
 return '<div class="cal-week"><div class="cal-week-grid">'+cols+'</div></div>';
}

function renderDayView(){
 const d=state.calDate,ts=getTrainingsForDay(d),dayName=DAY_NAMES[d.getDay()],hours=Array.from({length:24},(_,i)=>i);
 return '<div class="cal-day-view"><div class="cal-day-title">'+esc(dayName)+'</div><div class="cal-day-body">'+hours.map(h=>{const label=String(h).padStart(2,'0')+':00',hts=ts.filter(t=>new Date(t.start_time).getHours()===h);return '<div class="cal-hour-row"><div class="cal-hour-label">'+label+'</div><div class="cal-hour-events">'+hts.map(t=>'<div class="cal-event '+esc(t.status)+'" style="padding:3px 10px;cursor:pointer;font-weight:600" onclick="showTrainingDetail('+Number(t.id)+')">'+esc(t.title)+(t.trainer_name?' · '+esc(t.trainer_name):'')+'</div>').join('')+'</div></div>';}).join('')+'</div></div>';
}

function calClickDay(y,m,day){
  state.calDate=new Date(y,m,day);state.calView='day';
  return reloadCalendar(state.user?.role==='client'?'client-calendar':'staff-calendar');
}

function showTrainingDetail(id){
 const t=state.trainings.find(x=>x.id===id);if(!t)return;const canEdit=['manager','admin'].includes(state.user?.role),canDelete=state.user?.role==='admin';document.getElementById('td-title').textContent=t.title;
 document.getElementById('td-body').innerHTML='<div><span style="color:var(--t2);min-width:100px;display:inline-block">Статус:</span>'+statusBadge(t.status)+'</div><div><span style="color:var(--t2);min-width:100px;display:inline-block">Начало:</span>'+esc(fmt(t.start_time))+'</div><div><span style="color:var(--t2);min-width:100px;display:inline-block">Конец:</span>'+esc(fmt(t.end_time))+'</div>'+(t.client_name?'<div><span style="color:var(--t2);min-width:100px;display:inline-block">Клиент:</span>'+esc(t.client_name)+'</div>':'')+(t.trainer_name?'<div><span style="color:var(--t2);min-width:100px;display:inline-block">Тренер:</span>'+esc(t.trainer_name)+'</div>':'')+(t.description?'<div><span style="color:var(--t2);min-width:100px;display:inline-block">Описание:</span>'+esc(t.description)+'</div>':'');
 let actions='';if(canEdit)actions+='<button class="btn btn-ghost btn-sm" onclick="openEditTraining('+Number(id)+')">✏️ Изменить</button>';if(canDelete)actions+='<button class="btn btn-danger btn-sm" onclick="deleteTraining('+Number(id)+')">🗑 Удалить</button>';if(state.user?.role==='client'&&t.status==='scheduled'&&(new Date(t.start_time).getTime()-Date.now())>=2*60*60*1000)actions+='<button class="btn btn-danger btn-sm" onclick="cancelMyTraining('+Number(id)+')">Отменить запись</button>';document.getElementById('td-actions').innerHTML=actions;openModal('modal-training-detail');
}

function openEditTraining(id){
  const t=state.trainings.find(x=>x.id===id);if(!t)return;
  state.editingTrainingId=id;closeModal('modal-training-detail');
  document.getElementById('et-title').value=t.title;
  document.getElementById('et-desc').value=t.description||'';
  document.getElementById('et-status').value=t.status;
  var dS=new Date(t.start_time);var dE=new Date(t.end_time);
  var pad=function(n){return String(n).padStart(2,'0');};
  document.getElementById('et-start').value=pad(dS.getDate())+'.'+pad(dS.getMonth()+1)+'.'+dS.getFullYear()+' '+pad(dS.getHours())+':'+pad(dS.getMinutes());
  document.getElementById('et-end').value=pad(dE.getDate())+'.'+pad(dE.getMonth()+1)+'.'+dE.getFullYear()+' '+pad(dE.getHours())+':'+pad(dE.getMinutes());
  openModal('modal-edit-training');
}

async function submitEditTraining(){
  const id=state.editingTrainingId;
  const t=state.trainings.find(x=>x.id===id);
  try{
    await PUT('/schedule/'+id,{
      trainer_id:t?.trainer_id||null,
      title:document.getElementById('et-title').value,
      description:document.getElementById('et-desc').value,
      start_time:parseRuDate(document.getElementById('et-start').value),
      end_time:parseRuDate(document.getElementById('et-end').value),
      status:document.getElementById('et-status').value,
    });
    closeModal('modal-edit-training');
    toast('Занятие обновлено');
    loadStaffSchedule();
    loadClients();
  }catch(e){toast(e.message,'error');}
}

async function deleteTraining(id){
  const overlay=document.createElement('div');
  overlay.style.cssText='position:fixed;inset:0;background:rgba(0,0,0,.72);z-index:2000;display:flex;align-items:center;justify-content:center';
  overlay.innerHTML='<div style="background:var(--bg2);border:1px solid var(--border);border-radius:var(--r3);padding:32px;width:380px;box-shadow:0 24px 64px rgba(0,0,0,.6)">'
    +'<h3 style="font-size:18px;font-weight:700;margin-bottom:10px">Удалить занятие?</h3>'
    +'<p style="font-size:14px;color:var(--t2);margin-bottom:28px">Это действие нельзя отменить.</p>'
    +'<div style="display:flex;gap:10px;justify-content:flex-end">'
    +'<button id="del-cancel" class="btn btn-ghost">Отмена</button>'
    +'<button id="del-confirm" class="btn btn-danger" style="background:var(--red);color:#fff;border:none">Удалить</button>'
    +'</div></div>';
  document.body.appendChild(overlay);
  document.getElementById('del-cancel').onclick=()=>overlay.remove();
  document.getElementById('del-confirm').onclick=async()=>{
    overlay.remove();
    try{
      await DELETE('/schedule/'+id);
      closeModal('modal-training-detail');
      toast('Занятие удалено');
      loadStaffSchedule();
    }catch(e){toast(e.message,'error');}
  };
}

let _calendarLoadVersion=0;
function calendarRange(){
  const d=state.calDate;let from,to;
  if(state.calView==='month'){from=new Date(d.getFullYear(),d.getMonth(),1);to=new Date(d.getFullYear(),d.getMonth()+1,0);}
  else if(state.calView==='week'){from=new Date(d);from.setDate(d.getDate()-(d.getDay()||7)+1);to=new Date(from);to.setDate(from.getDate()+6);}
  else{from=new Date(d);to=new Date(d);}
  // The API's day boundaries are UTC; include neighbouring dates for local time zones.
  from.setDate(from.getDate()-1);to.setDate(to.getDate()+1);
  return '?date_from='+dateParam(from)+'&date_to='+dateParam(to);
}
function reloadCalendar(cid){return cid==='client-calendar'?loadClientSchedule():loadStaffSchedule();}
async function loadCalendar(cid,isStaff){
  const version=++_calendarLoadVersion,token=state.token;
  state.trainings=[];renderCalendar(cid);
  try{
    const [trainings,requests]=await Promise.all([GET('/schedule'+calendarRange()),isStaff?GET('/schedule/requests'):Promise.resolve(null)]);
    if(version!==_calendarLoadVersion||token!==state.token)return;
    state.trainings=trainings||[];renderCalendar(cid);
    if(isStaff)renderPendingRequests(requests||[]);
  }catch(e){if(version===_calendarLoadVersion&&token===state.token)toast(e.message,'error');}
}
async function loadClientSchedule(){return loadCalendar('client-calendar',false);}

let _schedStatusFilter = 'all';

function schedFilter(status, btn) {
  _schedStatusFilter = status;
  document.querySelectorAll('[id^="sched-filter-"]').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  renderCalendar('client-calendar');
}

async function openTrainingRequestModalWithTrainer(trainerId){
  await openTrainingRequestModal();
  setReqType('trainer');
  const select=document.getElementById('req-trainer');
  select.value=trainerId;
}

async function openTrainingRequestModal(){
  const select=document.getElementById('req-trainer');
  select.innerHTML='<option value="">Любой тренер</option>';
  try{
    const trainers=await GET('/trainers');
    (trainers||[]).forEach(t=>{
      const opt=document.createElement('option');
      opt.value=t.id;opt.textContent=t.full_name+' — '+specLabel(t.specialization);
      select.appendChild(opt);
    });
  }catch{}

  // Минимальная дата — сегодня
  const today=dateParam(new Date());
  document.getElementById('req-date').min=today;
  document.getElementById('req-date').value=today;

  // Сетка времён
  const times=['07:00','08:00','09:00','10:00','11:00','12:00','13:00','14:00','15:00','16:00','17:00','18:00','19:00','20:00','21:00'];
  let selectedTime=null;
  const grid=document.getElementById('req-time-grid');
  grid.innerHTML=times.map(t=>'<button type="button" onclick="selectTime(this,\''+t+'\')" style="padding:8px 4px;background:var(--bg4);border:1px solid var(--border);border-radius:var(--r);color:var(--t2);font-size:13px;font-weight:600;cursor:pointer;transition:all .2s" onmouseover="if(!this.classList.contains(\'time-sel\'))this.style.borderColor=\'var(--blue)\'" onmouseout="if(!this.classList.contains(\'time-sel\'))this.style.borderColor=\'var(--border)\'">'+t+'</button>').join('');

  setReqType('solo');
  openModal('modal-req-training');
}

function selectTime(btn, time){
  document.querySelectorAll('#req-time-grid button').forEach(b=>{
    b.classList.remove('time-sel');
    b.style.background='var(--bg4)';
    b.style.borderColor='var(--border)';
    b.style.color='var(--t2)';
  });
  btn.classList.add('time-sel');
  btn.style.background='var(--blue)';
  btn.style.borderColor='var(--blue)';
  btn.style.color='#fff';
  btn.dataset.time=time;
}

async function submitTrainingRequest(){
 hideErr('req-err');if(_reqType==='solo'){try{const preferred=new Date(Date.now()+5*60*1000);await POST('/schedule/requests',{preferred_at:preferred.toISOString(),comment:'Индивидуальная тренировка, ближайшее доступное время. '+(document.getElementById('req-comment').value||'')});closeModal('modal-req-training');document.getElementById('req-comment').value='';toast('Заявка отправлена!');}catch(e){showErr('req-err',e.message);}return;}
 const date=document.getElementById('req-date').value,timeBtn=document.querySelector('#req-time-grid button.time-sel'),time=timeBtn?timeBtn.dataset.time:'',preferred_at=date&&time?new Date(date+'T'+time+':00').toISOString():'',comment=document.getElementById('req-comment').value,trainerVal=document.getElementById('req-trainer').value;
 if(!date){showErr('req-err','Выберите дату');return;}if(!time){showErr('req-err','Выберите время');return;}if(new Date(preferred_at)<=new Date()){showErr('req-err','Выберите время в будущем');return;}const trainer_id=trainerVal?Number(trainerVal):null;
 try{await POST('/schedule/requests',{preferred_at,trainer_id,comment});closeModal('modal-req-training');document.getElementById('req-comment').value='';toast('Заявка отправлена!');}catch(e){showErr('req-err',e.message);}
}

// STAFF UI
function populateStaffUI(){
  const u=state.user;if(!u)return;
  const ini=initials(u.full_name);
  document.getElementById('staff-avatar').textContent=ini;
  document.getElementById('staff-big-avatar').textContent=ini;
  document.getElementById('staff-fullname-display').textContent=u.full_name;
  document.getElementById('sp-fullname').value=u.full_name;
  document.getElementById('sp-phone').value=u.phone;
  document.getElementById('sp-email').value=u.email;
  setStaffProfileEditMode(u.role!=='admin');
  const exportBtn=document.getElementById('export-clients-btn');if(exportBtn)exportBtn.style.display=u.role==='admin'?'':'none';
  const finNav=document.getElementById('snav-finance');
  if(finNav)finNav.style.display=u.role==='admin'?'flex':'none';
  const dashNav=document.getElementById('snav-dashboard');
  if(dashNav)dashNav.style.display=u.role==='admin'?'flex':'none';
  const addBtn=document.getElementById('add-trainer-btn');
  if(addBtn)addBtn.style.display=u.role==='admin'?'inline-flex':'none';
  const navName=document.getElementById('staff-nav-name');
  const navRole=document.getElementById('staff-nav-role');
  if(navName)navName.textContent=u.full_name.split(' ').slice(0,2).join(' ');
  if(navRole)navRole.textContent=roleLabel(u.role);
  const btnApproveApp=document.getElementById('btn-approve-app');
  const btnRejectApp=document.getElementById('btn-reject-app');
  if(btnApproveApp)btnApproveApp.onclick=confirmApproveApp;
  if(btnRejectApp)btnRejectApp.onclick=confirmRejectApp;
  // Сразу загружаем бейджи
  (async()=>{
    try{
      const [requests,apps,tasks]=await Promise.all([
        GET('/schedule/requests'),
        GET('/applications?status=pending'),
        GET('/crm/tasks')
      ]);
      updateNavBadge('snav-schedule',(requests||[]).length);
      updateNavBadge('snav-applications',(apps||[]).length);
      updateNavBadge('snav-tasks',(tasks||[]).filter(t=>t.status==='open').length);
    }catch{}
  })();
  startPolling();
}

let _pollingInterval=null;
function startPolling(){
  if(_pollingInterval)clearInterval(_pollingInterval);
  _pollingInterval=setInterval(async()=>{
    if(!state.user||!state.token){clearInterval(_pollingInterval);return;}
    try{
      const requests=await GET('/schedule/requests');
      const apps=await GET('/applications?status=pending');
      const tasks=await GET('/crm/tasks');
      const newReqs=(requests||[]).length;
      const newApps=(apps||[]).length;
      updateNavBadge('snav-tasks',(tasks||[]).filter(t=>t.status==='open').length);
      updateNavBadge('snav-schedule',newReqs);
      updateNavBadge('snav-applications',newApps);
      if(state.currentStaffSection==='schedule')renderPendingRequests(requests||[]);
      if(state.currentStaffSection==='applications'){state.applications=apps||[];renderApplications();}
      if(state.currentStaffSection==='trainers'){
        const trainers=await GET(staffTrainersPath());
        state.trainers=trainers||[];
        renderStaffTrainers();
    }
      const prevReqs=parseInt(localStorage.getItem('_prevReqs')||'0');
      const prevApps=parseInt(localStorage.getItem('_prevApps')||'0');
    if(newReqs>prevReqs)toast('Новая заявка на тренировку (+'+(newReqs-prevReqs)+')');
    if(newApps>prevApps)toast('Новая заявка на регистрацию (+'+(newApps-prevApps)+')');
      localStorage.setItem('_prevReqs',newReqs);localStorage.setItem('_prevApps',newApps);
    }catch{}
  },15000);
}

function staffTrainerFilter(spec, btn){
  document.querySelectorAll('#staff-trainer-filters .pay-tab').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  renderStaffTrainers();
}

function updateNavBadge(navId,count){
  const nav=document.getElementById(navId);if(!nav)return;
  const old=nav.querySelector('.nav-badge');if(old)old.remove();
  if(count>0){
    const badge=document.createElement('span');badge.className='nav-badge';badge.textContent=count;
    badge.style.cssText='margin-left:auto;background:var(--red);color:#fff;border-radius:20px;padding:1px 7px;font-size:11px;font-weight:700';
    nav.appendChild(badge);
  }
}

function staffNav(sec){
  const sections=['dashboard','tasks','schedule','clients','applications','trainers','shop','finance','profile'];
  sections.forEach(s=>{
    const el=document.getElementById('ssec-'+s);if(el)el.style.display='none';
    const nav=document.getElementById('snav-'+s);if(nav)nav.classList.remove('active');
  });
  const el=document.getElementById('ssec-'+sec);
if(el){
  el.style.display = sec === 'schedule' ? 'block' : 'block';
  el.style.animation='none';
  el.offsetHeight;
  el.style.animation='fadeUp .25s ease';
}
  const nav=document.getElementById('snav-'+sec);if(nav)nav.classList.add('active');
  state.currentStaffSection=sec;
  if(sec==='schedule')loadStaffSchedule();
  if(sec==='clients')loadClients();
  if(sec==='applications')loadApplications();
  if(sec==='trainers')loadStaffTrainers();
  if(sec==='shop')loadStaffShop();
  if(sec==='finance')loadFinance();
  if(sec==='dashboard')loadCRMDashboard();
  if(sec==='tasks')loadTasks(false);
  if(sec==='profile'&&state.user?.role==='admin')cancelStaffProfileEdit(false);
}

async function loadStaffSchedule(){return loadCalendar('staff-calendar',true);}

function renderPendingRequests(list){
 const existing=document.getElementById('pending-requests-block');if(existing)existing.remove();if(!list.length)return;const block=document.createElement('div');block.id='pending-requests-block';block.style.cssText='padding:0 24px 24px';
 let html='<div style="font-family:var(--f2);font-size:18px;font-weight:700;margin-bottom:12px">Заявки на тренировку <span style="background:var(--blue);color:#fff;border-radius:20px;padding:2px 10px;font-size:13px">'+list.length+'</span></div><div style="display:flex;flex-direction:column;gap:8px">';
 list.forEach(r=>{html+='<div style="background:var(--bg3);border:1px solid var(--border);border-radius:var(--r2);padding:14px 18px;display:flex;align-items:center;justify-content:space-between;gap:12px"><div><div style="font-weight:700;margin-bottom:4px">'+esc(r.client_name||'—')+'</div><div style="font-size:13px;color:var(--t2)">Желаемое время: '+esc(fmt(r.preferred_at))+'</div>'+(r.comment?'<div style="font-size:13px;color:var(--t3);margin-top:3px">'+esc(r.comment)+'</div>':'')+'</div><div style="display:flex;gap:8px;flex-shrink:0"><button class="btn btn-success btn-sm" onclick="openApproveRequestModal('+Number(r.id)+')">Принять</button><button class="btn btn-danger btn-sm" onclick="rejectTrainingRequest('+Number(r.id)+')">Отклонить</button></div></div>';});html+='</div>';block.innerHTML=html;document.getElementById('ssec-schedule').appendChild(block);
}

async function rejectTrainingRequest(id){
  const overlay=document.createElement('div');
  overlay.style.cssText='position:fixed;inset:0;background:rgba(0,0,0,.72);z-index:2000;display:flex;align-items:center;justify-content:center';
  overlay.innerHTML='<div style="background:var(--bg2);border:1px solid var(--border);border-radius:var(--r3);padding:28px;width:360px;box-shadow:0 24px 64px rgba(0,0,0,.6)">'
    +'<div style="font-size:32px;text-align:center;margin-bottom:14px">🚫</div>'
    +'<h3 style="text-align:center;font-size:17px;font-weight:700;margin-bottom:20px">Отклонить заявку на тренировку?</h3>'
    +'<div style="display:flex;gap:10px">'
    +'<button id="rtr-cancel" class="btn btn-ghost" style="flex:1">Отмена</button>'
    +'<button id="rtr-confirm" class="btn btn-danger" style="flex:1">Отклонить</button>'
    +'</div></div>';
  document.body.appendChild(overlay);
  document.getElementById('rtr-cancel').onclick=()=>overlay.remove();
  document.getElementById('rtr-confirm').onclick=async()=>{
    overlay.remove();
    try{await POST('/schedule/requests/'+id+'/reject');toast('Заявка отклонена');loadStaffSchedule();}
    catch(e){toast(e.message,'error');}
  };
}

// APPROVE REQUEST MODAL
let _pendingReqId=null;

async function openApproveRequestModal(reqId){
  _pendingReqId=Number(reqId);
  try{
    const reqs=await GET('/schedule/requests');
    const req=(reqs||[]).find(function(r){return Number(r.id)===Number(reqId);});
    if(!req)return;
    const trainers=await GET('/trainers'),select=document.getElementById('ar-trainer');
    select.replaceChildren(new Option('Без тренера',''));
    (trainers||[]).forEach(t=>select.add(new Option(t.full_name,String(t.id))));
    if(req.trainer_id && !(trainers||[]).some(t=>Number(t.id)===Number(req.trainer_id))){
      select.add(new Option((req.trainer_name||'Выбранный тренер')+' — недоступен',String(req.trainer_id)));
    }
    select.value=req.trainer_id?String(req.trainer_id):'';
    document.getElementById('approve-req-info').innerHTML='<strong>'+esc(req.client_name)+'</strong><br>Желаемое время: '+esc(fmt(req.preferred_at))+'<br>'+esc(req.comment||'');
    document.getElementById('ar-title').value='Персональная тренировка';
    document.getElementById('ar-desc').value=req.comment||'';
    var d=new Date(req.preferred_at);
    var pad=function(n){return String(n).padStart(2,'0');};
    var dateStr=pad(d.getDate())+'.'+pad(d.getMonth()+1)+'.'+d.getFullYear();
    var startH=d.getHours();
    var endDate=new Date(d.getTime()+60*60*1000);
    document.getElementById('ar-start').value=dateStr+' '+pad(startH)+':'+pad(d.getMinutes());
    document.getElementById('ar-end').value=pad(endDate.getDate())+'.'+pad(endDate.getMonth()+1)+'.'+endDate.getFullYear()+' '+pad(endDate.getHours())+':'+pad(endDate.getMinutes());
    hideErr('ar-err');
    document.getElementById('btn-approve-req').onclick=submitApproveRequest;
    document.getElementById('btn-reject-req').onclick=submitRejectRequest;
    openModal('modal-approve-req');
  }catch(e){toast('Ошибка загрузки заявки','error');}
}

async function submitApproveRequest(){
  hideErr('ar-err');
  var start=document.getElementById('ar-start').value;
  var end=document.getElementById('ar-end').value;
  if(!start||!end){showErr('ar-err','Укажите время начала и конца');return;}
  if(!_pendingReqId){showErr('ar-err','Ошибка: ID заявки не найден');return;}
  try{
    var reqs=await GET('/schedule/requests');
    var req=(reqs||[]).find(function(r){return Number(r.id)===Number(_pendingReqId);});
    var body={
      client_id:req?req.client_id:null,
      trainer_id:Number(document.getElementById('ar-trainer').value)||null,
      clear_trainer:!document.getElementById('ar-trainer').value,
      title:document.getElementById('ar-title').value,
      description:document.getElementById('ar-desc').value,
      start_time:parseRuDate(start),
      end_time:parseRuDate(end),
    };
    await POST('/schedule/requests/'+_pendingReqId+'/approve',body);
    closeModal('modal-approve-req');
    toast('Заявка принята, занятие создано');
    loadStaffSchedule();
    const requests=await GET('/schedule/requests');
    updateNavBadge('snav-schedule',(requests||[]).length);
  }catch(e){showErr('ar-err',e.message||'Ошибка');}
}

async function submitRejectRequest(){
  try{
    await POST('/schedule/requests/'+_pendingReqId+'/reject');
    closeModal('modal-approve-req');
    toast('Заявка отклонена');
    loadStaffSchedule();
    const requests=await GET('/schedule/requests');
    updateNavBadge('snav-schedule',(requests||[]).length);
  }catch(e){toast(e.message||'Ошибка','error');}
}

async function openCreateTrainingModal(){
  hideErr('ctr-err');
  const trainerSel=document.getElementById('ctr-trainer');
  trainerSel.innerHTML='<option value="">Без тренера</option>';
  document.getElementById('ctr-client-search').value='';
  document.getElementById('ctr-client').value='';
  document.getElementById('ctr-client-dropdown').style.display='none';
  _ctrClients=[];
  try{
    const [users,trainers]=await Promise.all([GET('/users'),GET('/trainers')]);
    _ctrClients=(users||[]).filter(u=>u.role==='client');
    (trainers||[]).forEach(t=>{
      const opt=document.createElement('option');
      opt.value=t.id;
      opt.textContent=t.full_name+' — '+specLabel(t.specialization);
      trainerSel.appendChild(opt);
    });
  }catch{}
  const now=new Date();
  const pad=n=>String(n).padStart(2,'0');
  const dateStr=pad(now.getDate())+'.'+pad(now.getMonth()+1)+'.'+now.getFullYear();
  document.getElementById('ctr-start').value=dateStr+' 10:00';
  document.getElementById('ctr-end').value=dateStr+' 11:00';
  document.getElementById('ctr-title').value='Персональная тренировка';
  document.getElementById('ctr-desc').value='';
  setCtrType('solo');
  openModal('modal-create-training');
}

async function submitCreateTraining(){
 hideErr('ctr-err');const client_id=parseInt(document.getElementById('ctr-client').value),title=document.getElementById('ctr-title').value.trim(),description=document.getElementById('ctr-desc').value;if(!client_id){showErr('ctr-err','Выберите клиента');return;}if(!title){showErr('ctr-err','Введите название');return;}
 try{if(_ctrType==='solo'){const end=new Date(),start=new Date(end.getTime()-1000);const created=await POST('/schedule',{client_id,trainer_id:null,title,description,start_time:start.toISOString(),end_time:end.toISOString(),status:'scheduled'});await PUT('/schedule/'+created.id,{trainer_id:null,title,description,start_time:start.toISOString(),end_time:end.toISOString(),status:'completed'});}
 else{const trainer_id=parseInt(document.getElementById('ctr-trainer').value)||null,start=document.getElementById('ctr-start').value,end=document.getElementById('ctr-end').value;if(!start||!end){showErr('ctr-err','Укажите время');return;}await POST('/schedule',{client_id,trainer_id,title,description,start_time:parseRuDate(start),end_time:parseRuDate(end),status:'scheduled'});}
 closeModal('modal-create-training');toast('Тренировка добавлена');loadStaffSchedule();loadClients();}catch(e){showErr('ctr-err',e.message);}
}

let _ctrType='solo';

function setCtrType(type){
  _ctrType=type;
  document.getElementById('ctr-type-solo').className=type==='solo'?'btn btn-primary btn-sm':'btn btn-ghost btn-sm';
  document.getElementById('ctr-type-trainer').className=type==='trainer'?'btn btn-primary btn-sm':'btn btn-ghost btn-sm';
  document.getElementById('ctr-trainer-block').style.display=type==='trainer'?'block':'none';
  document.getElementById('ctr-time-block').style.display=type==='trainer'?'block':'none';
  if(type==='solo'){
    const now=new Date();
    const pad=n=>String(n).padStart(2,'0');
    const timeStr=pad(now.getHours())+':'+pad(now.getMinutes());
    const dateStr=pad(now.getDate())+'.'+pad(now.getMonth()+1)+'.'+now.getFullYear();
    let info=document.getElementById('ctr-solo-time-info');
    if(!info){
      info=document.createElement('div');
      info.id='ctr-solo-time-info';
      info.style.cssText='font-size:13px;color:var(--t2);background:var(--bg4);border-radius:var(--r);padding:10px 14px;border:1px solid var(--border)';
      document.getElementById('ctr-trainer-block').parentElement.insertBefore(info,document.getElementById('ctr-trainer-block'));
    }
    info.style.display='block';
    info.textContent='Время тренировки: '+dateStr+' '+timeStr+' — будет отмечена как завершённая';
  } else {
    const info=document.getElementById('ctr-solo-time-info');
    if(info)info.style.display='none';
  }
}

// CLIENTS
async function loadClients(){
  const tb=document.getElementById('clients-tbody');
  if(tb) tb.innerHTML='<tr>'+Array(8).fill('<td><div class="skeleton" style="height:18px;margin:4px 0"></div></td>').join('')+'</tr>'.repeat(5);
  try{
    const [users,trainers]=await Promise.all([GET('/users'),GET('/trainers')]);
    state.clients=users||[];
    state.trainers=trainers||[];
    renderClientsTable();
  }catch(e){
    toast(e.message||'Ошибка загрузки клиентов','error');
  }
}

let clientsPage=1;const CLIENTS_PER_PAGE=15;

function toggleClientMenu(id){
  const menu=document.getElementById('cmenu-'+id);if(!menu)return;
  const isOpen=menu.style.display!=='none';closeAllClientMenus();if(!isOpen)menu.style.display='block';
}
function closeAllClientMenus(){document.querySelectorAll('[id^="cmenu-"]').forEach(m=>m.style.display='none');}

function renderClientsTable(){
  const search=(document.getElementById('client-search')?.value||'').toLowerCase();
  const role=document.getElementById('client-role-filter')?.value||'';
  let list=state.clients.filter(c=>{
    const matchSearch=!search||c.full_name.toLowerCase().includes(search)||c.phone.includes(search);
    const matchRole=!role||c.role===role;
    const isTrainer=state.trainers.some(t=>t.user_id===c.id);
    return matchSearch&&matchRole&&!isTrainer;
  });
  const total=list.length;
  const totalPages=Math.ceil(total/CLIENTS_PER_PAGE)||1;
  if(clientsPage>totalPages)clientsPage=1;
  const start=(clientsPage-1)*CLIENTS_PER_PAGE;
  const paged=list.slice(start,start+CLIENTS_PER_PAGE);
  const tb=document.getElementById('clients-tbody');
  if(!paged.length){tb.innerHTML='<tr><td colspan="9" style="text-align:center;color:var(--t3);padding:24px">Клиентов нет</td></tr>';}
  else{
    tb.innerHTML=paged.map(c=>{
      const isAdmin=state.user?.role==='admin';
      const activeColor=c.is_active?'#f87171':'#4ade80';
      const activeLabel=c.is_active?'Заблокировать':'Активировать';
      const statusHtml=c.password_setup_required?'<span class="badge badge-yellow">Ожидает активации</span>':(c.is_active?'<span class="badge badge-green">Активен</span>':'<span class="badge badge-red">Заблокирован</span>');
      const activationBtn=c.password_setup_required?'<button onclick="closeAllClientMenus();resendActivation('+c.id+')" style="width:100%;padding:8px 10px;background:none;border:none;color:var(--yellow);font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'var(--bg4)\'" onmouseout="this.style.background=\'none\'">Повторить приглашение</button>':'';
      const adminBtns=isAdmin&&!c.password_setup_required
        ?'<div style="height:1px;background:var(--border);margin:4px 0"></div>'
          +'<button onclick="closeAllClientMenus();toggleClient('+c.id+','+c.is_active+')" style="width:100%;padding:8px 10px;background:none;border:none;color:'+activeColor+';font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'rgba(239,68,68,.08)\'" onmouseout="this.style.background=\'none\'">'+activeLabel+'</button>'
          +'<div style="height:1px;background:var(--border);margin:4px 0"></div>'
          +'<button onclick="closeAllClientMenus();deleteClient('+c.id+')" style="width:100%;padding:8px 10px;background:none;border:none;color:#f87171;font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'rgba(239,68,68,.08)\'" onmouseout="this.style.background=\'none\'">Деактивировать</button>'
        :'';
      return '<tr>'
        +'<td><div style="display:flex;align-items:center;gap:9px">'
        +'<div class="avatar" style="width:30px;height:30px;font-size:11px">'+esc(initials(c.full_name))+'</div>'
        +'<div><div style="font-weight:600">'+esc(c.full_name)+'</div><div style="font-size:12px;color:var(--t3)">'+esc(roleLabel(c.role))+'</div></div>'
        +'</div></td>'
        +'<td>'+esc(c.phone)+'</td>'
        +'<td style="font-size:13px">'+esc(c.email)+'</td>'
        +'<td style="font-weight:700;color:var(--blue2)">'+esc(fmtMoney(c.balance))+' ₽</td>'
        +'<td>'+esc(c.visits)+'</td>'
        +'<td>'+(c.sessions_left!=null?'<span style="font-weight:700;color:var(--green)">'+esc(c.sessions_left)+' зан.</span>':'<span style="font-size:12px;color:var(--t3)">Нет абонемента</span>')+'</td>'
        +'<td style="font-size:12px;color:var(--t2)">'+esc(c.last_visit_at?fmtDate(c.last_visit_at):'—')+'</td>'
        +'<td>'+statusHtml+'</td>'
        +'<td><div style="position:relative;display:inline-block">'
        +'<button onclick="toggleClientMenu('+c.id+')" style="background:var(--bg4);border:1px solid var(--border);border-radius:var(--r);color:var(--t2);padding:6px 12px;cursor:pointer;font-size:18px;font-weight:700;letter-spacing:3px" onmouseover="this.style.borderColor=\'var(--blue)\'" onmouseout="this.style.borderColor=\'var(--border)\'">•••</button>'
        +'<div id="cmenu-'+c.id+'" style="display:none;position:absolute;right:0;top:calc(100% + 6px);background:var(--bg2);border:1px solid var(--border);border-radius:var(--r2);padding:6px;min-width:170px;z-index:300;box-shadow:0 8px 32px rgba(0,0,0,.5)">'
        +'<button onclick="closeAllClientMenus();openClientCRM('+c.id+')" style="width:100%;padding:8px 10px;background:none;border:none;color:var(--green);font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'var(--bg4)\'" onmouseout="this.style.background=\'none\'">CRM-карточка</button>'
        +'<button onclick="closeAllClientMenus();openEditUserModal('+c.id+')" style="width:100%;padding:8px 10px;background:none;border:none;color:var(--t1);font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'var(--bg4)\'" onmouseout="this.style.background=\'none\'">Редактировать</button>'
        +activationBtn
        +'<button onclick="closeAllClientMenus();openStaffTopUpModal('+Number(c.id)+')" style="width:100%;padding:8px 10px;background:none;border:none;color:var(--blue2);font-size:13px;font-weight:600;cursor:pointer;border-radius:var(--r);text-align:left" onmouseover="this.style.background=\'var(--bg4)\'" onmouseout="this.style.background=\'none\'">Пополнить баланс</button>'
        +adminBtns
        +'</div></div></td></tr>';
    }).join('');
  }
  let pg=document.getElementById('clients-pagination');
  if(!pg){
    pg=document.createElement('div');pg.id='clients-pagination';
    pg.style.cssText='display:flex;align-items:center;justify-content:space-between;padding:12px 16px;border-top:1px solid var(--border)';
    tb.closest('table').parentElement.closest('.card')?.appendChild(pg);
  }
  const pages=Array.from({length:totalPages},(_,i)=>i+1)
    .filter(p=>p===1||p===totalPages||Math.abs(p-clientsPage)<=1)
    .reduce((acc,p,i,arr)=>{if(i>0&&arr[i-1]!==p-1)acc.push('...');acc.push(p);return acc;},[]);
  pg.innerHTML='<span style="font-size:13px;color:var(--t2)">Показано '+Math.min(start+1,total)+'–'+Math.min(start+CLIENTS_PER_PAGE,total)+' из '+total+'</span>'
    +'<div style="display:flex;gap:4px">'
    +'<button class="btn btn-ghost btn-sm" onclick="clientsPage=1;renderClientsTable()" '+(clientsPage===1?'disabled':'')+'>«</button>'
    +'<button class="btn btn-ghost btn-sm" onclick="clientsPage--;renderClientsTable()" '+(clientsPage===1?'disabled':'')+'>‹</button>'
    +pages.map(p=>p==='...'?'<span style="padding:6px 8px;color:var(--t3)">...</span>':'<button class="btn btn-sm '+(p===clientsPage?'btn-primary':'btn-ghost')+'" onclick="clientsPage='+p+';renderClientsTable()">'+p+'</button>').join('')
    +'<button class="btn btn-ghost btn-sm" onclick="clientsPage++;renderClientsTable()" '+(clientsPage===totalPages?'disabled':'')+'>›</button>'
    +'<button class="btn btn-ghost btn-sm" onclick="clientsPage='+totalPages+';renderClientsTable()" '+(clientsPage===totalPages?'disabled':'')+'>»</button>'
    +'</div>';
}

async function toggleClient(id,isActive){
  try{await PATCH('/users/'+id+'/'+(isActive?'deactivate':'activate'));toast(isActive?'Заблокирован':'Активирован');loadClients();}
  catch(e){toast(e.message,'error');}
}

async function deleteClient(id){if(!confirm('Деактивировать пользователя? История тренировок и платежей сохранится.'))return;try{await DELETE('/users/'+id);toast('Пользователь деактивирован');loadClients();}catch(e){toast(e.message,'error');}}

function updateCreateUserPasswordMode(){
  if(state.editingClientId)return;
  const role=document.getElementById('cu-role').value,wrap=document.getElementById('cu-pass-wrap'),hint=document.getElementById('cu-pass-hint');
  const isClient=role==='client';wrap.style.display=isClient?'none':'';if(hint)hint.style.display=isClient?'':'none';
  document.getElementById('cu-submit-btn').textContent=isClient?'Создать и отправить приглашение':'Создать';
}

function openCreateUserModal(){['cu-name','cu-phone','cu-email','cu-pass'].forEach(id=>{const el=document.getElementById(id);if(el)el.value='';});state.editingClientId=null;const role=document.getElementById('cu-role');role.disabled=state.user?.role==='manager';role.value='client';document.getElementById('cu-gender').value='male';document.getElementById('cu-modal-title').textContent='Добавить пользователя';document.getElementById('cu-modal-subtitle').style.display='none';hideErr('cu-err');updateCreateUserPasswordMode();openModal('modal-create-user');}

function openEditUserModal(id){const c=state.clients.find(x=>x.id===id);if(!c)return;if(state.user?.role==='manager'&&c.role!=='client'){toast('Менеджер может редактировать только клиентов','error');return;}document.getElementById('cu-name').value=c.full_name;document.getElementById('cu-phone').value=c.phone;document.getElementById('cu-email').value=c.email;document.getElementById('cu-pass').value='';const role=document.getElementById('cu-role');role.value=c.role;role.disabled=true;document.getElementById('cu-gender').value=c.gender||'male';document.getElementById('cu-pass-wrap').style.display=state.user?.role==='admin'&&!c.password_setup_required?'':'none';document.getElementById('cu-modal-title').textContent='Редактировать пользователя';document.getElementById('cu-modal-subtitle').textContent=c.full_name;document.getElementById('cu-modal-subtitle').style.display='block';document.getElementById('cu-submit-btn').textContent='Сохранить';state.editingClientId=id;hideErr('cu-err');openModal('modal-create-user');}

async function submitCreateUser(){
  const full_name=document.getElementById('cu-name').value.trim(),phone=document.getElementById('cu-phone').value.trim(),email=document.getElementById('cu-email').value.trim(),password=document.getElementById('cu-pass').value,gender=document.getElementById('cu-gender').value;let role=document.getElementById('cu-role').value;hideErr('cu-err');if(state.user?.role==='manager')role='client';
  try{
    let message='Сохранено',messageType='success';
    if(state.editingClientId){
      await PUT('/users/'+state.editingClientId,{full_name,phone,email,gender});
      if(password){if(state.user?.role!=='admin')throw new Error('Сбрасывать пароль может только администратор');await PUT('/users/'+state.editingClientId+'/password',{new_password:password});}
    }else if(role==='client'){
      const result=await POST('/users/invite',{full_name,phone,email,gender});message=result?.email_sent?'Клиент создан. Приглашение отправлено на email.':'Клиент создан, но письмо не отправлено. Используйте «Повторить приглашение».';messageType=result?.email_sent?'success':'error';
    }else{
      if(password.length<8)throw new Error('Пароль должен быть не короче 8 символов');await POST('/users',{full_name,phone,email,password,role,gender});
    }
    closeModal('modal-create-user');state.editingClientId=null;toast(message,messageType);loadClients();
  }catch(e){showErr('cu-err',e.message);}
}

// APPLICATIONS
async function loadApplications(status='pending'){
  try{const data=await GET('/applications?status='+status);state.applications=data||[];renderApplications();}catch{}
}

function appFilter(status,btn){
  document.querySelectorAll('#ssec-applications .pay-tab').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');loadApplications(status);
}

let appsPage = 1;
const APPS_PER_PAGE = 10;

function renderApplications(){
 const list=document.getElementById('applications-list');if(!state.applications.length){list.innerHTML='<p style="color:var(--t2);text-align:center;padding:32px">Заявок нет</p>';return;}const total=state.applications.length,totalPages=Math.ceil(total/APPS_PER_PAGE)||1;if(appsPage>totalPages)appsPage=1;const start=(appsPage-1)*APPS_PER_PAGE,paged=state.applications.slice(start,start+APPS_PER_PAGE);
 list.innerHTML=paged.map(a=>'<div class="app-card"><div><div style="font-weight:700;margin-bottom:4px">'+esc(a.full_name)+'</div><div style="font-size:13px;color:var(--t2)">'+esc(a.phone)+' · '+esc(a.email)+'</div><div style="font-size:12px;color:var(--t3);margin-top:4px">'+esc(fmtDate(a.created_at))+'</div></div><div style="display:flex;align-items:center;gap:8px">'+statusBadge(a.status==='pending'?'pending':a.status==='approved'?'completed':'cancelled')+(a.status==='pending'?'<button class="btn btn-success btn-sm" onclick="openApproveApp('+Number(a.id)+')">Принять</button><button class="btn btn-danger btn-sm" onclick="quickRejectApp('+Number(a.id)+')">Отклонить</button>':'')+'</div></div>').join('');
 let pg=document.getElementById('apps-pagination');if(!pg){pg=document.createElement('div');pg.id='apps-pagination';pg.style.cssText='display:flex;align-items:center;justify-content:space-between;padding:16px 0;margin-top:12px';list.parentElement.appendChild(pg);}pg.innerHTML='<span style="font-size:13px;color:var(--t2)">Показано '+Math.min(start+1,total)+'–'+Math.min(start+APPS_PER_PAGE,total)+' из '+total+'</span><div style="display:flex;gap:4px"><button class="btn btn-ghost btn-sm" onclick="appsPage--;renderApplications()" '+(appsPage===1?'disabled':'')+'>‹</button>'+Array.from({length:totalPages},(_,i)=>i+1).map(p=>'<button class="btn btn-sm '+(p===appsPage?'btn-primary':'btn-ghost')+'" onclick="appsPage='+p+';renderApplications()">'+p+'</button>').join('')+'<button class="btn btn-ghost btn-sm" onclick="appsPage++;renderApplications()" '+(appsPage===totalPages?'disabled':'')+'>›</button></div>';
}

function openApproveApp(id){
  const a=state.applications.find(x=>x.id===Number(id));if(!a)return;
  state.approvingAppId=Number(id);
  document.getElementById('app-info').innerHTML='<strong>'+esc(a.full_name)+'</strong><br>'+esc(a.phone)+' · '+esc(a.email);
  document.getElementById('app-gender').value=a.gender||'';
  hideErr('app-err');openModal('modal-approve-app');
}

async function confirmApproveApp(){
  hideErr('app-err');if(!state.approvingAppId){showErr('app-err','Ошибка ID');return;}
  try{
    const gender=document.getElementById('app-gender').value;
    if(!gender){showErr('app-err','Укажите пол клиента');return;}
    const result=await POST('/applications/'+state.approvingAppId+'/approve',{gender});
    closeModal('modal-approve-app');
    toast(result?.email_sent?'Клиент принят. Ссылка для создания пароля отправлена на email.':'Клиент принят, но письмо не отправлено. Используйте «Повторить приглашение».',result?.email_sent?'success':'error');
    loadApplications();loadClients();const apps=await GET('/applications?status=pending');updateNavBadge('snav-applications',(apps||[]).length);
  }catch(e){showErr('app-err',e.message);}
}

async function resendActivation(id){
  try{const result=await POST('/users/'+id+'/resend-activation',{});toast(result?.email_sent?'Новое приглашение отправлено на email':'Ссылка создана, но письмо не отправлено',result?.email_sent?'success':'error');}
  catch(e){toast(e.message||'Не удалось отправить приглашение','error');}
}

async function confirmRejectApp(){
  try{
    await POST('/applications/'+state.approvingAppId+'/reject');
    closeModal('modal-approve-app');toast('Заявка отклонена');loadApplications();
  }catch(e){toast(e.message,'error');}
}

async function quickRejectApp(id){
  const a=state.applications.find(x=>x.id===Number(id));
  const name=a?a.full_name:'клиента';
  const overlay=document.createElement('div');
  overlay.style.cssText='position:fixed;inset:0;background:rgba(0,0,0,.72);z-index:2000;display:flex;align-items:center;justify-content:center';
  overlay.innerHTML='<div style="background:var(--bg2);border:1px solid var(--border);border-radius:var(--r3);padding:28px;width:380px;box-shadow:0 24px 64px rgba(0,0,0,.6)">'
    +'<div style="font-size:32px;text-align:center;margin-bottom:14px">🚫</div>'
    +'<h3 style="text-align:center;font-size:17px;font-weight:700;margin-bottom:8px">Отклонить заявку?</h3>'
    +'<p style="text-align:center;font-size:14px;color:var(--t2);margin-bottom:24px">'+esc(name)+'</p>'
    +'<div style="display:flex;gap:10px">'
    +'<button id="qr-cancel" class="btn btn-ghost" style="flex:1">Отмена</button>'
    +'<button id="qr-confirm" class="btn btn-danger" style="flex:1">Отклонить</button>'
    +'</div></div>';
  document.body.appendChild(overlay);
  document.getElementById('qr-cancel').onclick=()=>overlay.remove();
  document.getElementById('qr-confirm').onclick=async()=>{
    overlay.remove();
    try{
      await POST('/applications/'+Number(id)+'/reject');
      toast('Отклонено');
      loadApplications();
      const apps=await GET('/applications?status=pending');
      updateNavBadge('snav-applications',(apps||[]).length);
    }
    catch(e){toast(e.message,'error');}
  };
}

// TRAINERS STAFF
function staffTrainersPath(){return state.user?.role==='admin'?'/admin/trainers':'/trainers';}
async function loadStaffTrainers(){
  const grid=document.getElementById('staff-trainers-grid');
  if(grid) grid.innerHTML=Array(3).fill('<div class="card" style="height:320px"><div class="skeleton" style="height:200px;border-radius:0"></div><div style="padding:18px;display:flex;flex-direction:column;gap:10px"><div class="skeleton" style="height:16px;width:70%"></div><div class="skeleton" style="height:12px;width:50%"></div><div class="skeleton" style="height:12px;width:90%"></div></div></div>').join('');
  try{const data=await GET(staffTrainersPath());state.trainers=data||[];renderStaffTrainers();}catch{}
}

function renderStaffTrainers(){
  const activeBtn=document.querySelector('#staff-trainer-filters .pay-tab.active'),specMap={'Рельеф тела':'body_relief','Похудение':'weight_loss','Набор массы':'mass_gain'},spec=specMap[activeBtn?.textContent.trim()]||'all';
  const filtered=spec==='all'?state.trainers:state.trainers.filter(t=>t.specialization===spec),grid=document.getElementById('staff-trainers-grid'),isAdmin=state.user?.role==='admin';
  if(!filtered.length){grid.innerHTML='<p style="color:var(--t2)">Тренеров нет</p>';return;}
  grid.innerHTML=filtered.map(t=>{
    const photo=safeImageURL(t.photo_url),id=Number(t.id);
    return '<div class="trainer-card"><div class="trainer-photo">'+(photo?'<img src="'+esc(photo)+'" style="width:100%;height:100%;object-fit:cover" alt="">':'💪')+'</div><div class="trainer-info"><div class="trainer-name">'+esc(t.full_name)+'</div>'+(!t.is_active?'<span class="badge badge-gray">Неактивен</span>':'')+'<div class="trainer-spec">'+esc(specLabel(t.specialization))+'</div><div class="trainer-bio">'+esc(t.bio||'')+'</div><div style="font-size:12px;color:var(--t3);margin-top:8px">Опыт: '+esc(t.experience_years)+' лет</div>'+(isAdmin?'<div style="display:flex;gap:8px;margin-top:10px"><button class="btn btn-ghost btn-sm" onclick="openEditTrainerModal('+id+')">Изменить</button>'+(t.is_active?'<button class="btn btn-danger btn-sm" onclick="deleteTrainer('+id+')">Деактивировать</button>':'<button class="btn btn-success btn-sm" onclick="restoreTrainer('+id+')">Восстановить</button>')+'</div>':'')+'</div></div>';
  }).join('');
}
async function restoreTrainer(id){
  const t=state.trainers.find(t=>t.id===id);if(!t)return;
  try{const updated=await PUT('/trainers/'+id,{specialization:t.specialization,bio:t.bio,photo_url:t.photo_url,experience_years:t.experience_years,is_active:true});if(!updated.is_active)throw new Error('Сначала активируйте учётную запись тренера');toast('Тренер восстановлен');await loadStaffTrainers();}catch(e){toast(e.message,'error');}
}

let _trainerPhotoBlob=null;
let _trainerPhotoObjectURL='';
let _trainerExistingPhotoURL='';
let _trainerPhotoRemove=false;

function trainerPhotoPlaceholder(){
 const preview=document.getElementById('ct-photo-preview');if(!preview)return;preview.replaceChildren();
 const ns='http://www.w3.org/2000/svg',svg=document.createElementNS(ns,'svg');svg.setAttribute('viewBox','0 0 24 24');svg.setAttribute('aria-hidden','true');
 const circle=document.createElementNS(ns,'circle');circle.setAttribute('cx','12');circle.setAttribute('cy','8');circle.setAttribute('r','4');
 const path=document.createElementNS(ns,'path');path.setAttribute('d','M4 21c0-4.4 3.6-8 8-8s8 3.6 8 8');svg.append(circle,path);preview.appendChild(svg);
}

function clearTrainerPhotoObjectURL(){if(_trainerPhotoObjectURL){URL.revokeObjectURL(_trainerPhotoObjectURL);_trainerPhotoObjectURL='';}}

function showTrainerPhotoPreview(src){
 const preview=document.getElementById('ct-photo-preview');if(!preview)return;preview.replaceChildren();
 const url=safeImageURL(src);if(!url&&!(src instanceof Blob)){trainerPhotoPlaceholder();return;}
 const img=document.createElement('img');img.alt='Фото тренера';img.style.cssText='width:100%;height:100%;object-fit:cover';
 if(src instanceof Blob){clearTrainerPhotoObjectURL();_trainerPhotoObjectURL=URL.createObjectURL(src);img.src=_trainerPhotoObjectURL;}else img.src=url;
 img.onerror=()=>trainerPhotoPlaceholder();preview.appendChild(img);
}

function resetTrainerPhotoState(existingURL=''){
 clearTrainerPhotoObjectURL();_trainerPhotoBlob=null;_trainerExistingPhotoURL=safeImageURL(existingURL)||'';_trainerPhotoRemove=false;
 const input=document.getElementById('ct-photo-file');if(input)input.value='';
 const remove=document.getElementById('ct-photo-remove');if(remove)remove.style.display=_trainerExistingPhotoURL?'inline-flex':'none';
 const meta=document.getElementById('ct-photo-meta');if(meta)meta.textContent='JPG, PNG, WebP. Перед загрузкой фото автоматически уменьшается до 320×320 и сжимается до WebP, обычно 15–50 КБ.';
 if(_trainerExistingPhotoURL)showTrainerPhotoPreview(_trainerExistingPhotoURL);else trainerPhotoPlaceholder();
}

function loadImageForCompression(file){
 return new Promise((resolve,reject)=>{const url=URL.createObjectURL(file),img=new Image();img.onload=()=>{URL.revokeObjectURL(url);resolve(img);};img.onerror=()=>{URL.revokeObjectURL(url);reject(new Error('Не удалось прочитать изображение'));};img.src=url;});
}

function canvasBlob(canvas,type,quality){return new Promise(resolve=>canvas.toBlob(resolve,type,quality));}

async function compressTrainerPhoto(file){
 if(!file||!String(file.type||'').startsWith('image/'))throw new Error('Выберите файл изображения');
 if(file.size>10*1024*1024)throw new Error('Исходное фото должно быть меньше 10 МБ');
 const img=await loadImageForCompression(file),side=Math.min(img.naturalWidth||img.width,img.naturalHeight||img.height);
 if(!side)throw new Error('Некорректное изображение');
 const sx=((img.naturalWidth||img.width)-side)/2,sy=((img.naturalHeight||img.height)-side)/2;
 const attempts=[
  [320,0.72],[320,0.60],[320,0.50],[288,0.52],[256,0.50],[224,0.46]
 ];
 let best=null;
 for(const [size,quality] of attempts){
  const canvas=document.createElement('canvas');canvas.width=size;canvas.height=size;const ctx=canvas.getContext('2d',{alpha:false});ctx.fillStyle='#111827';ctx.fillRect(0,0,size,size);ctx.drawImage(img,sx,sy,side,side,0,0,size,size);
  let blob=await canvasBlob(canvas,'image/webp',quality);if(!blob||blob.type!=='image/webp')blob=await canvasBlob(canvas,'image/jpeg',quality);
  if(!blob)continue;best=blob;if(blob.size<=64*1024)break;
 }
 if(!best)throw new Error('Не удалось сжать изображение');
 if(best.size>96*1024)throw new Error('Фото не удалось достаточно сжать. Выберите другое изображение');
 return best;
}

async function handleTrainerPhotoFile(input){
 hideErr('ct-err');const file=input.files?.[0];if(!file)return;
 const meta=document.getElementById('ct-photo-meta');if(meta)meta.textContent='Сжимаем фото…';
 try{_trainerPhotoBlob=await compressTrainerPhoto(file);_trainerPhotoRemove=false;showTrainerPhotoPreview(_trainerPhotoBlob);const remove=document.getElementById('ct-photo-remove');if(remove)remove.style.display='inline-flex';if(meta)meta.textContent='Готово: '+Math.max(1,Math.round(_trainerPhotoBlob.size/1024))+' КБ. На сервер отправится только сжатая версия.';}
 catch(e){_trainerPhotoBlob=null;input.value='';if(meta)meta.textContent=e.message;showErr('ct-err',e.message);if(_trainerExistingPhotoURL)showTrainerPhotoPreview(_trainerExistingPhotoURL);else trainerPhotoPlaceholder();}
}

function removeTrainerPhotoSelection(){
 clearTrainerPhotoObjectURL();_trainerPhotoBlob=null;_trainerPhotoRemove=true;const input=document.getElementById('ct-photo-file');if(input)input.value='';trainerPhotoPlaceholder();const remove=document.getElementById('ct-photo-remove');if(remove)remove.style.display='none';const meta=document.getElementById('ct-photo-meta');if(meta)meta.textContent='Фото будет удалено после сохранения.';
}

async function uploadTrainerPhoto(trainerID,blob){
 const form=new FormData(),ext=blob.type==='image/jpeg'?'jpg':'webp';form.append('photo',blob,'trainer.'+ext);
 const headers={};if(state.token)headers.Authorization='Bearer '+state.token;
 let r;try{r=await fetch(API+'/trainers/'+encodeURIComponent(trainerID)+'/photo',{method:'POST',headers,body:form});}catch{throw new Error('Не удалось загрузить фото тренера');}
 const data=await r.json().catch(()=>null);if(!r.ok)throw new Error(data?.error||('Ошибка загрузки фото: HTTP '+r.status));return data;
}

async function openCreateTrainerModal(){
 state.editingTrainerId=null;state.editingTrainerActive=true;document.getElementById('ct-userid').closest('.inp-wrap').style.display='';['ct-exp','ct-bio'].forEach(id=>{const el=document.getElementById(id);if(el)el.value='';});resetTrainerPhotoState('');hideErr('ct-err');
 const select=document.getElementById('ct-userid');select.replaceChildren(new Option('Загрузка...',''));
 try{const [users,trainers]=await Promise.all([GET('/users'),GET('/admin/trainers')]),assigned=new Set((trainers||[]).map(t=>Number(t.user_id)));select.replaceChildren();(users||[]).filter(u=>u.is_active&&!assigned.has(Number(u.id))).forEach(u=>select.add(new Option(u.full_name+' ('+roleLabel(u.role)+')',String(u.id))));if(!select.options.length)select.add(new Option('Нет доступных пользователей',''));}catch{select.replaceChildren(new Option('Ошибка загрузки',''));}
 document.getElementById('ct-modal-title').textContent='Добавить тренера';document.getElementById('ct-submit-btn').textContent='Добавить';openModal('modal-create-trainer');
}

async function submitCreateTrainer(){
 hideErr('ct-err');const base={specialization:document.getElementById('ct-spec').value,bio:document.getElementById('ct-bio').value.trim(),experience_years:parseInt(document.getElementById('ct-exp').value)||0};
 const btn=document.getElementById('ct-submit-btn'),wasEditing=!!state.editingTrainerId;btn.disabled=true;const oldText=btn.textContent;btn.textContent='Сохраняем…';
 try{
  let trainer;
  if(wasEditing){trainer=await PUT('/trainers/'+state.editingTrainerId,{...base,photo_url:_trainerExistingPhotoURL,is_active:state.editingTrainerActive!==false});}
  else{const userID=parseInt(document.getElementById('ct-userid').value);if(!userID)throw new Error('Выберите пользователя');trainer=await POST('/trainers',{user_id:userID,...base,photo_url:''});state.editingTrainerId=trainer.id;document.getElementById('ct-userid').closest('.inp-wrap').style.display='none';}
  if(_trainerPhotoRemove&&!_trainerPhotoBlob&&wasEditing){btn.textContent='Удаляем фото…';trainer=await DELETE('/trainers/'+trainer.id+'/photo');}
  if(_trainerPhotoBlob){btn.textContent='Загружаем фото…';trainer=await uploadTrainerPhoto(trainer.id,_trainerPhotoBlob);}
  toast(wasEditing?'Тренер обновлён':'Тренер назначен');state.editingTrainerId=null;state.editingTrainerActive=true;document.getElementById('ct-userid').closest('.inp-wrap').style.display='';closeModal('modal-create-trainer');resetTrainerPhotoState('');await loadStaffTrainers();
 }catch(e){if(!wasEditing&&state.editingTrainerId)showErr('ct-err','Тренер уже создан. '+e.message+' Нажмите «Добавить» ещё раз, чтобы повторить загрузку фото.');else showErr('ct-err',e.message);}finally{btn.disabled=false;btn.textContent=oldText;}
}

async function openEditTrainerModal(id){
 const t=state.trainers.find(x=>x.id===id);if(!t)return;state.editingTrainerId=id;state.editingTrainerActive=t.is_active;document.getElementById('ct-spec').value=t.specialization;document.getElementById('ct-exp').value=t.experience_years;document.getElementById('ct-bio').value=t.bio||'';resetTrainerPhotoState(t.photo_url||'');document.getElementById('ct-modal-title').textContent='Изменить тренера';document.getElementById('ct-submit-btn').textContent='Сохранить';document.getElementById('ct-userid').closest('.inp-wrap').style.display='none';hideErr('ct-err');openModal('modal-create-trainer');
}

async function deleteTrainer(id){const t=state.trainers.find(x=>x.id===id);if(!confirm('Деактивировать тренера '+(t?.full_name||'')+'? История тренировок сохранится.'))return;try{await DELETE('/trainers/'+id);toast('Тренер деактивирован');loadStaffTrainers();}catch(e){toast(e.message,'error');}}

// SHOP STAFF
async function loadStaffShop(){try{const data=await GET('/shop/products?is_active=true');state.products=data||[];renderShopGrid(state.products,'staff-shop-grid',true);}catch(e){toast(e.message,'error');}}

function openCreateProductModal(){
  if(_productSaving)return;resetProductPhotoState();
  state.editingProductId=null;
  document.getElementById('cp-modal-title').textContent='Добавить товар';
  document.getElementById('cp-submit-btn').textContent='Сохранить';
  ['cp-name','cp-price','cp-days','cp-sessions','cp-desc'].forEach(id=>{const el=document.getElementById(id);if(el)el.value='';});
  document.getElementById('cp-cat').value='subscription';document.getElementById('cp-subtype').value='monthly';toggleSubFields();hideErr('cp-err');openModal('modal-create-product');
}

function openEditProductModal(p){
  if(_productSaving)return;resetProductPhotoState(p.photo_url||'');
  state.editingProductId=p.id;
  document.getElementById('cp-modal-title').textContent='Изменить товар';
  document.getElementById('cp-name').value=p.name;
  document.getElementById('cp-price').value=p.price;
  document.getElementById('cp-cat').value=p.category;
  document.getElementById('cp-subtype').value=p.sub_type||'monthly';
  document.getElementById('cp-desc').value=p.description||'';
  document.getElementById('cp-days').value=p.duration_days||'';
  document.getElementById('cp-sessions').value=p.sessions_count||'';
  toggleSubFields();hideErr('cp-err');openModal('modal-create-product');
}

function toggleSubFields(){
  const cat=document.getElementById('cp-cat').value;
  document.getElementById('sub-fields').style.display=cat==='subscription'?'block':'none';
}

let _productSaving=false,_productPhotoBusy=false,_productPhotoVersion=0;
let _productPhotoBlob=null,_productPhotoPreview='',_productPhotoExisting='',_productPhotoRemoved=false;
function renderProductPhotoPreview(){
  const src=_productPhotoPreview||(_productPhotoRemoved?'':safeImageURL(_productPhotoExisting));
  document.getElementById('cp-photo-preview').innerHTML=src?'<img src="'+esc(src)+'" alt="Фото товара">':'<span>Нет фото</span>';
  document.getElementById('cp-photo-remove').style.display=src?'inline-flex':'none';
}
function resetProductPhotoState(existing=''){
  _productPhotoVersion++;_productPhotoBusy=false;_productPhotoBlob=null;_productPhotoRemoved=false;_productPhotoExisting=existing;
  if(_productPhotoPreview)URL.revokeObjectURL(_productPhotoPreview);_productPhotoPreview='';
  document.getElementById('cp-photo-file').value='';document.getElementById('cp-submit-btn').disabled=false;
  document.getElementById('cp-photo-meta').textContent='JPG, PNG или WebP. Фото появится в каталоге.';renderProductPhotoPreview();
}
async function compressProductPhoto(file){
  if(!['image/jpeg','image/png','image/webp'].includes(file.type))throw new Error('Выберите фото в формате JPG, PNG или WebP.');
  if(file.size>10*1024*1024)throw new Error('Выберите фото размером до 10 МБ.');
  const img=await loadImageForCompression(file),width=img.naturalWidth||img.width,height=img.naturalHeight||img.height;
  if(!width||!height||width*height>40000000)throw new Error('Изображение слишком большое или повреждено.');
  for(const [size,quality] of [[960,.85],[960,.7],[960,.55],[768,.6],[640,.5]]){
    const scale=Math.min(1,size/Math.max(width,height)),canvas=document.createElement('canvas');
    canvas.width=Math.max(1,Math.round(width*scale));canvas.height=Math.max(1,Math.round(height*scale));
    const ctx=canvas.getContext('2d');ctx.fillStyle='#ffffff';ctx.fillRect(0,0,canvas.width,canvas.height);ctx.drawImage(img,0,0,canvas.width,canvas.height);
    const blob=await canvasBlob(canvas,'image/jpeg',quality);if(blob&&blob.type==='image/jpeg'&&blob.size<=256*1024)return blob;
  }
  throw new Error('Не удалось подготовить фото. Выберите другое изображение.');
}
async function handleProductPhotoFile(input){
  if(_productSaving)return;const file=input.files?.[0];if(!file)return;
  const version=++_productPhotoVersion;_productPhotoBusy=true;document.getElementById('cp-submit-btn').disabled=true;hideErr('cp-err');
  document.getElementById('cp-photo-meta').textContent='Подготавливаем фото…';
  try{
    const blob=await compressProductPhoto(file);if(version!==_productPhotoVersion)return;
    if(_productPhotoPreview)URL.revokeObjectURL(_productPhotoPreview);
    _productPhotoBlob=blob;_productPhotoPreview=URL.createObjectURL(blob);_productPhotoRemoved=false;renderProductPhotoPreview();
    document.getElementById('cp-photo-meta').textContent='Фото готово. Нажмите «Сохранить».';
  }catch(e){if(version===_productPhotoVersion){showErr('cp-err',e.message);document.getElementById('cp-photo-meta').textContent='Выберите другое фото.';input.value='';}}
  finally{if(version===_productPhotoVersion){_productPhotoBusy=false;document.getElementById('cp-submit-btn').disabled=false;}}
}
function removeProductPhotoSelection(){
  if(_productSaving)return;_productPhotoVersion++;_productPhotoBusy=false;_productPhotoBlob=null;_productPhotoRemoved=true;
  if(_productPhotoPreview)URL.revokeObjectURL(_productPhotoPreview);_productPhotoPreview='';document.getElementById('cp-photo-file').value='';
  document.getElementById('cp-submit-btn').disabled=false;document.getElementById('cp-photo-meta').textContent='Фото будет убрано после сохранения.';renderProductPhotoPreview();
}
async function uploadProductPhoto(id,blob){
  const form=new FormData();form.append('photo',blob,'product.jpg');
  const response=await fetch(API+'/shop/products/'+id+'/photo',{method:'POST',headers:{Authorization:'Bearer '+state.token},body:form});
  const data=await response.json().catch(()=>null);if(!response.ok)throw new Error(data?.error||'Не удалось загрузить фото.');return data;
}
async function submitCreateProduct(){
  if(_productSaving||_productPhotoBusy)return;hideErr('cp-err');
  const cat=document.getElementById('cp-cat').value;
  const body={name:document.getElementById('cp-name').value.trim(),description:document.getElementById('cp-desc').value.trim(),price:parseFloat(document.getElementById('cp-price').value),category:cat,
    ...(cat==='subscription'?{sub_type:document.getElementById('cp-subtype').value,duration_days:parseInt(document.getElementById('cp-days').value)||null,sessions_count:parseInt(document.getElementById('cp-sessions').value)||null}:{}),};
  _productSaving=true;['cp-submit-btn','cp-photo-file','cp-photo-remove'].forEach(id=>document.getElementById(id).disabled=true);
  let productSaved=false;
  try{
    if(state.editingProductId)await PUT('/shop/products/'+state.editingProductId,body);
    else{const created=await POST('/shop/products',body);state.editingProductId=created.id;document.getElementById('cp-modal-title').textContent='Изменить товар';}
    productSaved=true;
    if(_productPhotoBlob)await uploadProductPhoto(state.editingProductId,_productPhotoBlob);
    else if(_productPhotoRemoved&&_productPhotoExisting)await DELETE('/shop/products/'+state.editingProductId+'/photo');
    resetProductPhotoState();closeModal('modal-create-product');toast('Сохранено');loadStaffShop();
  }catch(e){showErr('cp-err',(productSaved?'Товар сохранён, но фото не обновлено. Нажмите «Сохранить», чтобы повторить. ':'')+(e.message||'Ошибка соединения.'));}
  finally{_productSaving=false;['cp-submit-btn','cp-photo-file','cp-photo-remove'].forEach(id=>document.getElementById(id).disabled=false);}
}

async function deleteProduct(id){if(state.user?.role!=='admin'){toast('Деактивировать товар может только администратор','error');return;}if(!confirm('Деактивировать товар? История покупок сохранится.'))return;try{await DELETE('/shop/products/'+id);toast('Товар деактивирован');loadStaffShop();}catch(e){toast(e.message,'error');}}

// FINANCE
async function loadFinance(){
  try{
    const [payments,summary]=await Promise.all([GET('/finance/payments'),GET('/finance/summary')]);
    state.payments=payments||[];state.financeSummary=summary;renderFinance();
  }catch{}
}

let financePage=1;const FINANCE_PER_PAGE=20;

function renderFinance(){
 const x=state.financeSummary||{};document.getElementById('fin-income').textContent=fmtMoney(x.total_income)+' ₽';document.getElementById('fin-expense').textContent=fmtMoney(x.total_expense)+' ₽';document.getElementById('fin-refund').textContent=fmtMoney(x.total_refund)+' ₽';document.getElementById('fin-net').textContent=fmtMoney(x.net_balance)+' ₽';const total=state.payments.length,totalPages=Math.ceil(total/FINANCE_PER_PAGE)||1;if(financePage>totalPages)financePage=1;const start=(financePage-1)*FINANCE_PER_PAGE,paged=state.payments.slice(start,start+FINANCE_PER_PAGE),tb=document.getElementById('finance-tbody');
 if(!paged.length)tb.innerHTML='<tr><td colspan="5" style="text-align:center;color:var(--t3);padding:24px">Нет данных</td></tr>';else tb.innerHTML=paged.map(p=>'<tr><td>'+esc(fmt(p.created_at))+'</td><td>'+esc(p.client_name||'—')+'</td><td style="font-weight:700;color:'+(p.operation_type==='income'?'var(--green)':'var(--red)')+'">'+esc(fmtMoney(p.amount))+' ₽</td><td>'+esc(svcLabel(p.service_type))+'</td><td>'+(p.operation_type==='income'?'<span class="badge badge-green">Доход</span>':p.operation_type==='refund'?'<span class="badge badge-yellow">Возврат</span>':'<span class="badge badge-red">Расход</span>')+'</td></tr>').join('');
 let pg=document.getElementById('finance-pagination');if(!pg){pg=document.createElement('div');pg.id='finance-pagination';pg.style.cssText='display:flex;align-items:center;justify-content:space-between;padding:12px 16px;border-top:1px solid var(--border)';tb.closest('table').parentElement.closest('.card')?.appendChild(pg);}pg.innerHTML='<span style="font-size:13px;color:var(--t2)">Показано '+Math.min(start+1,total)+'–'+Math.min(start+FINANCE_PER_PAGE,total)+' из '+total+'</span><div style="display:flex;gap:4px"><button class="btn btn-ghost btn-sm" onclick="financePage=1;renderFinance()" '+(financePage===1?'disabled':'')+'>«</button><button class="btn btn-ghost btn-sm" onclick="financePage--;renderFinance()" '+(financePage===1?'disabled':'')+'>‹</button><button class="btn btn-ghost btn-sm" onclick="financePage++;renderFinance()" '+(financePage===totalPages?'disabled':'')+'>›</button><button class="btn btn-ghost btn-sm" onclick="financePage='+totalPages+';renderFinance()" '+(financePage===totalPages?'disabled':'')+'>»</button></div>';
}

async function openStaffTopUpModal(preselectID=null){const select=document.getElementById('staff-topup-client');select.replaceChildren(new Option('Загрузка...',''));document.getElementById('staff-topup-amount').value='';hideErr('staff-topup-err');try{const users=await GET('/users');select.replaceChildren();(users||[]).filter(u=>u.role==='client'&&u.is_active).forEach(u=>select.add(new Option(u.full_name+' ('+fmtMoney(u.balance)+' ₽)',String(u.id))));if(!select.options.length)select.add(new Option('Клиентов нет',''));if(preselectID&&Array.from(select.options).some(o=>Number(o.value)===Number(preselectID)))select.value=String(preselectID);}catch{select.replaceChildren(new Option('Ошибка загрузки',''));}openModal('modal-staff-topup');}

async function submitStaffTopUp(){
  const client_id = parseInt(document.getElementById('staff-topup-client').value);
  const amount    = parseFloat(document.getElementById('staff-topup-amount').value);
  hideErr('staff-topup-err');
  if(!client_id || !amount){ showErr('staff-topup-err','Заполните поля'); return; }
  try{
    await POST('/finance/topup',{client_id, amount});
    closeModal('modal-staff-topup');
    toast('Баланс пополнен');
    loadFinance();
    loadClients();
  }catch(e){ showErr('staff-topup-err', e.message); }
}

function setStaffProfileEditMode(editing){
  const isAdmin=state.user?.role==='admin';
  const editable=!isAdmin||Boolean(editing);
  ['sp-fullname','sp-phone','sp-email'].forEach(id=>{
    const input=document.getElementById(id);
    if(!input)return;
    input.readOnly=!editable;
    input.setAttribute('aria-readonly',String(!editable));
    input.classList.toggle('staff-profile-locked',!editable);
  });
  const editBtn=document.getElementById('staff-profile-edit-btn');
  const saveBtn=document.getElementById('staff-profile-save-btn');
  const cancelBtn=document.getElementById('staff-profile-cancel-btn');
  if(editBtn)editBtn.style.display=isAdmin&&!editable?'inline-flex':'none';
  if(saveBtn)saveBtn.style.display=editable?'inline-flex':'none';
  if(cancelBtn)cancelBtn.style.display=isAdmin&&editable?'inline-flex':'none';
}

function enableStaffProfileEdit(){
  if(state.user?.role!=='admin')return;
  const msg=document.getElementById('staff-profile-msg');
  if(msg)msg.style.display='none';
  setStaffProfileEditMode(true);
  document.getElementById('sp-fullname')?.focus();
}

function cancelStaffProfileEdit(clearMessage=true){
  if(!state.user)return;
  document.getElementById('sp-fullname').value=state.user.full_name||'';
  document.getElementById('sp-phone').value=state.user.phone||'';
  document.getElementById('sp-email').value=state.user.email||'';
  if(clearMessage){const msg=document.getElementById('staff-profile-msg');if(msg)msg.style.display='none';}
  setStaffProfileEditMode(state.user.role!=='admin');
}

async function saveStaffProfile(){
  if(state.user?.role==='admin'&&document.getElementById('sp-fullname')?.readOnly){
    return;
  }
  const full_name=document.getElementById('sp-fullname').value.trim();
  const phone=document.getElementById('sp-phone').value.trim();
  const email=document.getElementById('sp-email').value.trim();
  const msg=document.getElementById('staff-profile-msg');msg.style.display='none';
  try{
    const u=await PUT('/users/me',{full_name,phone,email,gender:state.user?.gender||'male'});
    state.user=u;saveAuth();
    populateStaffUI();
    msg.textContent='Данные сохранены';msg.style.color='var(--green)';msg.style.display='block';
  }catch(e){msg.textContent=e.message;msg.style.color='var(--red)';msg.style.display='block';}
}

// PUBLIC TRAINERS
async function loadTrainersPublic(){
  try{const data=await GET('/trainers');state.trainers=data||[];renderTrainers();}
  catch(e){
    state.trainers=[];
    const grid=document.getElementById('trainers-grid');
    if(grid){grid.replaceChildren();const p=document.createElement('p');p.style.cssText='color:var(--red);padding:20px';p.textContent='Не удалось загрузить тренеров: '+e.message;grid.appendChild(p);}
  }
}

function filterTrainers(spec,btn){
  document.querySelectorAll('.filter-btn').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');state.trainerFilter=spec;renderTrainers();
}

function renderTrainers(){
 const search=(document.getElementById('trainer-search')?.value||'').toLowerCase(),spec=state.trainerFilter,list=state.trainers.filter(t=>(spec==='all'||t.specialization===spec)&&(!search||String(t.full_name||'').toLowerCase().includes(search))),grid=document.getElementById('trainers-grid');if(!grid)return;if(!list.length){grid.innerHTML='<p style="color:var(--t2);padding:20px">Тренеров не найдено</p>';return;}
 grid.innerHTML=list.map(t=>{const photo=safeImageURL(t.photo_url);return '<div class="trainer-card"><div class="trainer-photo" style="background:linear-gradient(135deg,#13131f,#1a1a2e)">'+(photo?'<img src="'+esc(photo)+'" style="width:100%;height:100%;object-fit:cover" alt="">':'<span style="font-size:60px">💪</span>')+'</div><div class="trainer-info"><div class="trainer-name">'+esc(t.full_name)+'</div><div class="trainer-spec">'+esc(specLabel(t.specialization))+'</div><div class="trainer-bio">'+esc(t.bio||'')+'</div><div style="font-size:12px;color:var(--t3);margin-top:8px">Опыт: '+esc(t.experience_years)+' лет</div></div></div>';}).join('');
}

// MINI CALENDAR
function renderMiniCal(){
  const wrap=document.getElementById('mini-cal');if(!wrap)return;
  const now=new Date();const y=now.getFullYear(),m=now.getMonth();
  const firstDow=new Date(y,m,1).getDay()||7;const daysInM=new Date(y,m+1,0).getDate();
  const cells=[];for(let i=1;i<firstDow;i++)cells.push('');for(let i=1;i<=daysInM;i++)cells.push(i);while(cells.length%7)cells.push('');
  wrap.innerHTML='<div style="text-align:center;font-weight:700;font-size:14px;margin-bottom:14px;color:var(--t2)">'+MONTHS[m]+' '+y+'</div>'
    +'<div style="display:grid;grid-template-columns:repeat(7,1fr);gap:3px">'
    +DAYS.map((d,i)=>'<div style="text-align:center;font-size:11px;font-weight:700;color:'+(i>=5?'var(--blue2)':'var(--t3)')+';padding:4px 0">'+d+'</div>').join('')
    +cells.map(d=>'<div style="text-align:center;font-size:13px;padding:5px 3px;border-radius:5px;border:1px solid var(--border);'+(d===now.getDate()?'background:var(--blue);color:#fff;font-weight:700;':'color:var(--t2);')+(d===''?'opacity:0;':'')+'">'+d+'</div>').join('')
    +'</div>';
}

// Загружаем тренеров на лендинге
async function loadLandTrainers(){
 try{const data=await GET('/trainers'),grid=document.getElementById('land-trainers-grid');if(!grid)return;const list=(data||[]).slice(0,3);if(!list.length){grid.innerHTML='<p style="color:var(--t2)">Тренеров нет</p>';return;}grid.innerHTML=list.map(t=>{const photo=safeImageURL(t.photo_url);return '<div class="card" style="overflow:hidden;transition:all .3s;cursor:pointer"><div style="height:180px;background:linear-gradient(135deg,var(--bg4),var(--bg3));display:flex;align-items:center;justify-content:center;overflow:hidden">'+(photo?'<img src="'+esc(photo)+'" style="width:100%;height:100%;object-fit:cover" alt="">':'<span style="font-size:64px">💪</span>')+'</div><div style="padding:18px"><div style="font-weight:700;margin-bottom:4px">'+esc(t.full_name)+'</div><div style="font-size:13px;color:var(--blue2);font-weight:600;margin-bottom:6px">'+esc(specLabel(t.specialization))+'</div><div style="font-size:13px;color:var(--t2)">'+esc(t.bio||'')+'</div></div></div>';}).join('');}catch{}
}
loadLandTrainers();

// CRM+ — operational modules. AI/RAG code below is intentionally unchanged.
let _progressClientID=null;
function numOrNull(id){const v=document.getElementById(id)?.value;return v===''||v==null?null:Number(v);}
function openProgressEntry(clientID=null){_progressClientID=clientID;['pr-weight','pr-fat','pr-waist','pr-chest','pr-hips','pr-comment'].forEach(id=>{const e=document.getElementById(id);if(e)e.value='';});openModal('modal-progress');}
async function saveProgressEntry(){const body={weight:numOrNull('pr-weight'),body_fat:numOrNull('pr-fat'),waist:numOrNull('pr-waist'),chest:numOrNull('pr-chest'),hips:numOrNull('pr-hips'),comment:document.getElementById('pr-comment').value.trim()};try{const path=_progressClientID?'/crm/clients/'+_progressClientID+'/progress':'/crm/progress';await POST(path,body);closeModal('modal-progress');toast('Измерение сохранено');if(_progressClientID){openClientCRM(_progressClientID);}else{loadProgress();}}catch(e){toast(e.message,'error');}}
async function loadProgress(){const tb=document.getElementById('progress-tbody');if(tb)tb.innerHTML='<tr><td colspan="7" style="padding:22px;text-align:center;color:var(--t2)">Загрузка...</td></tr>';try{const rows=await GET('/crm/progress');renderProgress(rows||[]);}catch(e){if(tb)tb.innerHTML='<tr><td colspan="7" style="padding:22px;text-align:center;color:var(--red)">'+esc(e.message)+'</td></tr>';}}
function renderProgress(rows){const tb=document.getElementById('progress-tbody'),sum=document.getElementById('progress-summary');if(!tb||!sum)return;const latest=rows[0]||{},oldest=rows[rows.length-1]||{};const val=(v,s='')=>v==null?'—':Number(v).toFixed(1)+s;const delta=(a,b)=>a==null||b==null?'—':((a-b)>0?'+':'')+(a-b).toFixed(1);sum.innerHTML='<div class="fin-card"><div class="fin-card-label">Текущий вес</div><div class="fin-card-val">'+val(latest.weight,' кг')+'</div></div><div class="fin-card"><div class="fin-card-label">Изменение веса</div><div class="fin-card-val net">'+delta(latest.weight,oldest.weight)+' кг</div></div><div class="fin-card"><div class="fin-card-label">Жир</div><div class="fin-card-val">'+val(latest.body_fat,' %')+'</div></div><div class="fin-card"><div class="fin-card-label">Измерений</div><div class="fin-card-val">'+rows.length+'</div></div>';tb.innerHTML=rows.length?rows.map(p=>'<tr><td>'+esc(fmt(p.recorded_at))+'</td><td>'+val(p.weight,' кг')+'</td><td>'+val(p.body_fat,' %')+'</td><td>'+val(p.waist,' см')+'</td><td>'+val(p.chest,' см')+'</td><td>'+val(p.hips,' см')+'</td><td>'+esc(p.comment||'')+'</td></tr>').join(''):'<tr><td colspan="7" style="padding:24px;text-align:center;color:var(--t3)">Пока нет измерений</td></tr>';}

async function loadNotifications(){const wrap=document.getElementById('notifications-list');if(!wrap)return;wrap.innerHTML='<div class="card" style="padding:18px;color:var(--t2)">Загрузка...</div>';try{const rows=await GET('/crm/notifications');const unread=(rows||[]).filter(n=>!n.is_read).length;updateNavBadge('cnav-notifications',unread);wrap.innerHTML=(rows||[]).length?rows.map(n=>'<div class="card" style="padding:16px;'+(!n.is_read?'border-color:var(--blue);':'')+'"><div style="display:flex;justify-content:space-between;gap:12px"><div><div style="font-weight:700;margin-bottom:5px">'+esc(n.title)+'</div><div style="font-size:13px;color:var(--t2);line-height:1.5">'+esc(n.message)+'</div><div style="font-size:11px;color:var(--t3);margin-top:8px">'+esc(fmt(n.created_at))+'</div></div>'+(!n.is_read?'<button class="btn btn-ghost btn-sm" onclick="markNotificationRead('+n.id+')">Прочитано</button>':'')+'</div></div>').join(''):'<div class="card" style="padding:22px;color:var(--t3);text-align:center">Уведомлений нет</div>';}catch(e){wrap.innerHTML='<div class="card" style="padding:18px;color:var(--red)">'+esc(e.message)+'</div>';}}
async function markNotificationRead(id){try{await PATCH('/crm/notifications/'+id+'/read');loadNotifications();}catch(e){toast(e.message,'error');}}
async function cancelMyTraining(id){if(!confirm('Отменить эту тренировку? Отмена доступна не позднее чем за 2 часа до начала.'))return;try{await POST('/schedule/'+id+'/cancel',{});closeModal('modal-training-detail');toast('Запись отменена');loadClientSchedule();}catch(e){toast(e.message,'error');}}

async function loadCRMDashboard(){const cards=document.getElementById('crm-dashboard-cards'),util=document.getElementById('trainer-utilization');if(!cards||!util)return;cards.innerHTML='<div class="fin-card">Загрузка...</div>';try{const d=await GET('/crm/dashboard');const items=[['Активные клиенты',d.active_clients],['Активные абонементы',d.active_subscriptions],['Заморожено',d.frozen_subscriptions],['Тренировки сегодня',d.trainings_today],['Клиенты риска',d.at_risk_clients],['Заканчиваются ≤ 7 дней',d.expiring_subscriptions],['Открытые задачи',d.open_tasks],['Доход за 30 дней',fmtMoney(d.revenue_30_days)+' ₽']];cards.innerHTML=items.map(([k,v])=>'<div class="fin-card"><div class="fin-card-label">'+esc(k)+'</div><div class="fin-card-val">'+esc(v)+'</div></div>').join('');util.innerHTML=(d.trainer_utilization||[]).length?'<div class="table-wrap"><table><thead><tr><th>Тренер</th><th>Запланировано/завершено</th><th>Завершено</th><th>Выполнение</th></tr></thead><tbody>'+(d.trainer_utilization||[]).map(t=>'<tr><td>'+esc(t.trainer_name)+'</td><td>'+t.scheduled_30d+'</td><td>'+t.completed_30d+'</td><td>'+Number(t.completion_rate||0).toFixed(0)+'%</td></tr>').join('')+'</tbody></table></div>':'<div style="color:var(--t3)">Нет данных по тренерам</div>';}catch(e){cards.innerHTML='<div class="fin-card" style="color:var(--red)">'+esc(e.message)+'</div>';}}

async function loadTasks(includeDone=false){document.getElementById('task-filter-open')?.classList.toggle('active',!includeDone);document.getElementById('task-filter-all')?.classList.toggle('active',includeDone);const wrap=document.getElementById('tasks-list');if(!wrap)return;wrap.innerHTML='<div class="card" style="padding:18px;color:var(--t2)">Загрузка...</div>';try{const rows=await GET('/crm/tasks?include_done='+includeDone);updateNavBadge('snav-tasks',(rows||[]).filter(t=>t.status==='open').length);wrap.innerHTML=(rows||[]).length?rows.map(t=>'<div class="card" style="padding:15px;'+(t.is_overdue?'border-color:var(--red);':'')+'"><div style="display:flex;justify-content:space-between;gap:12px;align-items:flex-start"><div><div style="font-weight:700">'+esc(t.title)+'</div><div style="font-size:12px;color:var(--t2);margin-top:4px">'+(t.client_name?'Клиент: '+esc(t.client_name)+' · ':'')+(t.due_at?'Срок: '+esc(fmt(t.due_at)):'Без срока')+(t.is_overdue?' · <span style="color:var(--red)">Просрочено</span>':'')+'</div>'+(t.description?'<div style="font-size:13px;color:var(--t2);margin-top:7px">'+esc(t.description)+'</div>':'')+'</div><div style="display:flex;gap:6px">'+(t.status==='open'?'<button class="btn btn-success btn-sm" onclick="setTaskStatus('+t.id+',\'done\')">Готово</button><button class="btn btn-ghost btn-sm" onclick="setTaskStatus('+t.id+',\'cancelled\')">Отменить</button>':'<span class="badge badge-gray">'+esc(t.status)+'</span>')+'</div></div></div>').join(''):'<div class="card" style="padding:22px;text-align:center;color:var(--t3)">Задач нет</div>';}catch(e){wrap.innerHTML='<div class="card" style="padding:18px;color:var(--red)">'+esc(e.message)+'</div>';}}
async function setTaskStatus(id,status){try{await api('PATCH','/crm/tasks/'+id,{status});toast('Задача обновлена');loadTasks(false);}catch(e){toast(e.message,'error');}}
async function openTaskModal(clientID=null){try{if(!state.clients.length&&state.user?.role!=='client'){const users=await GET('/users');state.clients=users||[];}const clientSel=document.getElementById('task-client'),ass=document.getElementById('task-assignee');clientSel.innerHTML='<option value="">Без привязки</option>'+state.clients.filter(u=>u.role==='client').map(u=>'<option value="'+u.id+'">'+esc(u.full_name)+'</option>').join('');if(clientID)clientSel.value=String(clientID);let staff=state.clients.filter(u=>['manager','admin'].includes(u.role));if(!staff.some(u=>u.id===state.user.id))staff.unshift(state.user);ass.innerHTML=staff.map(u=>'<option value="'+u.id+'">'+esc(u.full_name)+'</option>').join('');ass.value=String(state.user.id);document.getElementById('task-title').value='';document.getElementById('task-desc').value='';document.getElementById('task-due').value='';openModal('modal-task');}catch(e){toast(e.message,'error');}}
async function saveTask(){const clientVal=document.getElementById('task-client').value,due=document.getElementById('task-due').value;const body={client_id:clientVal?Number(clientVal):null,assignee_id:Number(document.getElementById('task-assignee').value),title:document.getElementById('task-title').value.trim(),description:document.getElementById('task-desc').value.trim(),due_at:due?new Date(due).toISOString():null};try{await POST('/crm/tasks',body);closeModal('modal-task');toast('Задача создана');loadTasks(false);}catch(e){toast(e.message,'error');}}

async function openClientCRM(clientID){try{const [card,products]=await Promise.all([GET('/crm/clients/'+clientID),GET('/shop/products')]);document.getElementById('crm-client-title').textContent=card.user.full_name;document.getElementById('crm-client-sub').textContent=card.user.phone+' · '+card.user.email+' · посещений: '+card.user.visits;const subs=(card.subscriptions||[]).map(s=>'<div class="card" style="padding:13px;margin-bottom:8px"><div style="display:flex;justify-content:space-between;gap:10px;align-items:center;flex-wrap:wrap"><div><b>'+esc(s.product_name||'Абонемент')+'</b><div style="font-size:12px;color:var(--t2);margin-top:4px">до '+esc(fmtDate(s.end_date))+' · занятий: '+(s.sessions_left==null?'∞':s.sessions_left)+' · '+subscriptionBadge(s)+'</div></div><div style="display:flex;gap:6px">'+(s.is_active?(s.frozen_at?'<button class="btn btn-success btn-sm" onclick="unfreezeSubscription('+s.id+','+clientID+')">Разморозить</button>':'<button class="btn btn-ghost btn-sm" onclick="freezeSubscription('+s.id+','+clientID+')">Заморозить</button>')+'<button class="btn btn-ghost btn-sm" onclick="extendSubscription('+s.id+','+clientID+')">Продлить</button>':'')+'</div></div></div>').join('')||'<div style="color:var(--t3)">Абонементов нет</div>';const notes=(card.notes||[]).map(n=>'<div style="padding:9px 0;border-bottom:1px solid var(--border)"><div style="font-size:13px">'+esc(n.note)+'</div><div style="font-size:11px;color:var(--t3);margin-top:3px">'+esc(n.author_name||'')+' · '+esc(fmt(n.created_at))+'</div></div>').join('')||'<div style="color:var(--t3)">Заметок нет</div>';const progress=(card.progress||[]).slice(0,5).map(p=>'<tr><td>'+esc(fmtDate(p.recorded_at))+'</td><td>'+(p.weight??'—')+'</td><td>'+(p.body_fat??'—')+'</td><td>'+esc(p.comment||'')+'</td></tr>').join('')||'<tr><td colspan="4" style="color:var(--t3)">Нет измерений</td></tr>';const opts=(products||[]).filter(p=>p.is_active!==false).map(p=>'<option value="'+p.id+'">'+esc(p.name)+' — '+fmtMoney(p.price)+' ₽</option>').join('');document.getElementById('crm-client-body').innerHTML='<div class="finance-summary"><div class="fin-card"><div class="fin-card-label">Баланс</div><div class="fin-card-val">'+fmtMoney(card.user.balance)+' ₽</div></div><div class="fin-card"><div class="fin-card-label">Посещения</div><div class="fin-card-val">'+card.user.visits+'</div></div><div class="fin-card"><div class="fin-card-label">Без визита</div><div class="fin-card-val">'+(card.days_inactive==null?'—':card.days_inactive+' дн.')+'</div></div><div class="fin-card"><div class="fin-card-label">Задачи</div><div class="fin-card-val">'+(card.tasks||[]).filter(t=>t.status==='open').length+'</div></div></div><div style="display:grid;grid-template-columns:1.1fr .9fr;gap:16px"><div><h3 style="margin-bottom:10px">Абонементы</h3>'+subs+'<div class="card" style="padding:14px;margin-top:12px"><h3 style="margin-bottom:10px">Продажа на стойке</h3><div style="display:flex;gap:8px"><select class="inp" id="crm-sale-product">'+opts+'</select><button class="btn btn-primary btn-sm" onclick="staffSellToClient('+clientID+')">Продать</button></div></div></div><div><h3 style="margin-bottom:10px">Внутренние заметки</h3><div class="card" style="padding:12px"><div id="crm-notes-list">'+notes+'</div><textarea class="inp" id="crm-note-input" rows="2" placeholder="Заметка для сотрудников" style="margin-top:10px"></textarea><button class="btn btn-primary btn-sm" style="margin-top:8px" onclick="addClientNote('+clientID+')">Добавить</button></div></div></div><div class="card" style="padding:14px;margin-top:16px"><div style="display:flex;justify-content:space-between;gap:8px;align-items:center"><h3>Прогресс</h3><button class="btn btn-ghost btn-sm" onclick="openProgressEntry('+clientID+')">+ Измерение</button></div><div class="table-wrap" style="margin-top:10px"><table><thead><tr><th>Дата</th><th>Вес</th><th>Жир %</th><th>Комментарий</th></tr></thead><tbody>'+progress+'</tbody></table></div></div><div style="display:flex;gap:8px;flex-wrap:wrap;margin-top:16px"><button class="btn btn-primary btn-sm" onclick="openTaskModal('+clientID+')">+ Задача</button><button class="btn btn-ghost btn-sm" onclick="sendClientNotification('+clientID+')">Уведомить клиента</button><button class="btn btn-ghost btn-sm" onclick="downloadProtected(\'/crm/clients/'+clientID+'/export.csv\',\'client-'+clientID+'-crm.csv\')">Экспорт CRM-карточки</button></div>';openModal('modal-client-crm');}catch(e){toast(e.message,'error');}}
async function addClientNote(clientID){const note=document.getElementById('crm-note-input').value.trim();if(!note)return;try{await POST('/crm/clients/'+clientID+'/notes',{note});toast('Заметка добавлена');openClientCRM(clientID);}catch(e){toast(e.message,'error');}}
async function freezeSubscription(id,clientID){if(!confirm('Заморозить абонемент? Будущие тренировки сначала нужно перенести или отменить.'))return;try{await POST('/crm/subscriptions/'+id+'/freeze',{});toast('Абонемент заморожен');openClientCRM(clientID);}catch(e){toast(e.message,'error');}}
async function unfreezeSubscription(id,clientID){try{await POST('/crm/subscriptions/'+id+'/unfreeze',{});toast('Абонемент разморожен');openClientCRM(clientID);}catch(e){toast(e.message,'error');}}
async function extendSubscription(id,clientID){const days=Number(prompt('На сколько дней продлить?', '7'));if(!days)return;try{await POST('/crm/subscriptions/'+id+'/extend',{days});toast('Абонемент продлён');openClientCRM(clientID);}catch(e){toast(e.message,'error');}}
async function staffSellToClient(clientID){const sel=document.getElementById('crm-sale-product');if(!sel?.value)return;try{await POST('/shop/staff-purchase',{client_id:clientID,product_id:Number(sel.value)});toast('Продажа оформлена');openClientCRM(clientID);loadClients();}catch(e){toast(e.message,'error');}}
async function sendClientNotification(clientID){const title=prompt('Заголовок уведомления','Сообщение от SFEDU');if(!title)return;const message=prompt('Текст уведомления');if(!message)return;try{await POST('/crm/notifications',{user_id:clientID,title,message});toast('Уведомление отправлено');}catch(e){toast(e.message,'error');}}
async function downloadProtected(path,name){try{const r=await fetch(API+path,{headers:state.token?{Authorization:'Bearer '+state.token}:{}});if(!r.ok){const d=await r.json().catch(()=>({}));throw new Error(d.error||'Ошибка экспорта');}const blob=await r.blob(),url=URL.createObjectURL(blob),a=document.createElement('a');a.href=url;a.download=name;document.body.appendChild(a);a.click();a.remove();URL.revokeObjectURL(url);}catch(e){toast(e.message,'error');}}

// AI
function toggleAI(force){if(!state.user||!state.token){openModal('modal-login');return;}const p=document.getElementById('ai-panel'),open=typeof force==='boolean'?force:p.style.display==='none';p.style.display=open?'flex':'none';if(open){restoreAIConversation();loadAIConversation();setTimeout(()=>document.getElementById('ai-inp')?.focus(),0);}}
async function sendAI(){
 if(state.aiSending)return;const inp=document.getElementById('ai-inp'),btn=document.getElementById('ai-send'),msg=inp.value.trim();if(!msg)return;if(!state.user||!state.token){openModal('modal-login');return;}
 state.aiSending=true;inp.disabled=true;btn.disabled=true;inp.value='';addAIMsg(msg,'user');
 try{const body={message:msg};if(state.aiConversationId)body.conversation_id=state.aiConversationId;const data=await POST('/ai/chat',body);state.aiConversationId=data.conversation_id;persistAIConversation();addAIMsg(data.reply||'Нет ответа','ai',data.sources||[]);}catch(e){addAIMsg('Не удалось получить ответ: '+e.message,'ai');inp.value=msg;}finally{state.aiSending=false;inp.disabled=false;btn.disabled=false;inp.focus();}
}
function addAIMsg(text,role,sources=[]){
 const wrap=document.getElementById('ai-messages'),holder=document.createElement('div');holder.className='ai-msg-wrap '+role;const el=document.createElement('div');el.className='ai-msg '+role;el.textContent=String(text||'');holder.appendChild(el);
 if(role==='ai'&&Array.isArray(sources)&&sources.length){const box=document.createElement('div');box.className='ai-sources';sources.forEach(item=>{const chip=document.createElement('span');chip.className='ai-source';chip.textContent=item?.title||'Источник';if(item?.source)chip.title=item.source;box.appendChild(chip);});holder.appendChild(box);}
 wrap.appendChild(holder);wrap.scrollTop=wrap.scrollHeight;
}

function resetAIMessages(){const wrap=document.getElementById('ai-messages');if(!wrap)return;wrap.replaceChildren();addAIMsg('Привет! Я ИИ Спортсмен — помогу разобраться с системой SFEDU.','ai');}
function newAIConversation(){state.aiConversationId=null;persistAIConversation();resetAIMessages();document.getElementById('ai-inp')?.focus();}
async function loadAIConversation(){if(!state.aiConversationId){resetAIMessages();return;}try{const items=await GET('/ai/conversations/'+state.aiConversationId),wrap=document.getElementById('ai-messages');wrap.replaceChildren();(items||[]).forEach(m=>addAIMsg(m.content,m.role==='assistant'?'ai':'user',m.sources||[]));if(!(items||[]).length)resetAIMessages();}catch{state.aiConversationId=null;persistAIConversation();resetAIMessages();}}

renderMiniCal();

// The interface now uses a single dark theme, including older saved sessions.
document.documentElement.setAttribute('data-theme','dark');
try{localStorage.removeItem('sfedu_theme');}catch{}
