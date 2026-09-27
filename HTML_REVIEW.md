# Проверка frontend

В проекте два интерфейса:

- `frontend/index.html` — основной сайт/CRM SFEDU;
- `examples/ai-chat/index.html` — технический demo AI-чата.

Основной интерфейс больше не является одним огромным HTML: CSS вынесен в `frontend/assets/app.css`, JavaScript — в `frontend/assets/app.js`.

## Основной frontend

Проверено:

- HTML parser;
- уникальность `id`;
- `node --check frontend/assets/app.js`;
- same-origin API base `/api/v1`;
- автоматическая Bearer JWT авторизация после обычного login;
- role-specific client/manager/admin экраны;
- экранирование динамических пользовательских строк перед `innerHTML`-шаблонами;
- отсутствие `eval` / `new Function`;
- CRM+ routes: progress, notifications, client notes/tasks, subscription lifecycle, desk sale, dashboard, audit;
- клиентская отмена тренировки;
- CSV exports;
- responsive CSS для основных сеток.

Автоматическая статическая проверка запускается так:

```bash
python3 scripts/verify_frontend.py
node --check frontend/assets/app.js
```

## AI demo

Технический `examples/ai-chat/index.html` оставлен отдельно и не нужен обычному пользователю. Основной сайт уже имеет встроенную панель «ИИ Спортсмен» и использует существующие AI/RAG endpoints без изменения AI backend в CRM+ релизе.

## Ограничение среды

Полноценный browser E2E против localhost в sandbox не выполнялся. После запуска на Mac рекомендуется пройти smoke-test ролей client / manager / admin в браузере и проверить адаптивность через DevTools.

## TESTED-UI visual polish

The existing SFEDU visual identity was deliberately preserved: dark/navy background, blue accent, Exo 2/Rajdhani typography, compact CRM density and the same information architecture.

Polish applied without changing business behavior:

- deeper navy surfaces with restrained radial lighting;
- softer borders/shadows and consistent 10–16px radii;
- stronger hierarchy for cards, KPIs and primary actions;
- improved sidebar/topbar depth and active states;
- cleaner tables with sticky headers;
- more readable calendar cells/events;
- refined shop/trainer cards;
- improved modal backdrop/surfaces;
- explicit `:focus-visible` keyboard states;
- mobile/tablet refinements;
- `prefers-reduced-motion` support;
- light theme variables adjusted to retain contrast.

The AI panel received only matching visual styling; its service/client contract was not changed.
