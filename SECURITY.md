# Security notes

## Ротация старых секретов обязательна

Исходный публичный репозиторий содержал `.env`. Если его значения когда-либо использовались, считайте их раскрытыми и замените:

1. `JWT_SECRET`;
2. пароль PostgreSQL;
3. пароль администратора;
4. любые API keys, если они когда-либо находились рядом с этим конфигом.

В финальной копии `.env` отсутствует; присутствует только `.env.example` с placeholders.

Удаление `.env` из текущего commit не очищает старую Git-историю. При продолжении старого repository history используйте `git filter-repo`/эквивалент и всё равно ротируйте секреты.


## Account activation / password reset

- постоянный пароль клиента не передаётся сотруднику и не отправляется по email;
- activation/reset tokens генерируются из 32 cryptographically-random bytes;
- в PostgreSQL хранится только SHA-256 token;
- tokens имеют назначение (`activation`/`password_reset`), TTL и `used_at`;
- новый token инвалидирует предыдущие неиспользованные tokens того же типа;
- после установки/сброса пароля token становится одноразово использованным;
- pending client account имеет `is_active=false` и `password_setup_required=true`;
- обычный admin activate не может обойти обязательную первичную установку пароля;
- forgot-password возвращает одинаковый публичный ответ для существующего и неизвестного email;
- одноразовый token удаляется из address bar frontend сразу после чтения ссылки;
- frontend задаёт `Referrer-Policy: no-referrer` через meta tag;
- `MAIL_MODE=log` допустим только для development: он печатает ссылку в backend log и не отправляет письмо;
- production запускается только с `MAIL_MODE=smtp`.

## Production

- TLS должен завершаться на доверенном reverse proxy/load balancer;
- PostgreSQL не публикуйте наружу без необходимости;
- секреты храните в secret manager/environment, не в Git;
- `OPENAI_API_KEY` должен существовать только на backend;
- публичные login/application endpoints имеют in-process IP limits, но за reverse proxy нужен дополнительный trusted ingress rate limit;
- AI chat имеет per-user limit;
- для настоящего интернет-эквайринга staff top-up нужно заменить интеграцией с платёжным провайдером и проверяемыми signed callbacks;
- изменяющие операции записываются в `audit_logs`; для production рекомендуется дополнительно отправлять audit/HTTP logs во внешнее неизменяемое хранилище;
- регулярно обновляйте Go/runtime/base images и зависимости и запускайте vulnerability scanning в CI.


## Дополнительное hardening в CRM+

- каждый HTTP-запрос получает безопасный `X-Request-ID`; он попадает в обычный лог и audit trail;
- mutating endpoints для финансов, расписания, пользователей, CRM и магазина обёрнуты audit middleware;
- добавлены `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy` и CSP;
- метрики `/metrics` не содержат персональных данных, но в публичном production-развёртывании endpoint лучше закрыть ingress-правилом для внутренней monitoring-сети;
- `/readyz` делает PostgreSQL обязательной dependency, а Redis остаётся optional;
- клиент не может сам начислить себе баланс; desk sale и top-up остаются staff actions;
- staff notes и follow-up tasks доступны только manager/admin.

CSRF-токены не добавлялись намеренно: текущая авторизация использует Bearer JWT в `Authorization`, а не cookie-based session. Если позже JWT будет перенесён в cookies, CSRF-защита станет обязательной.

2FA для персонала и внешний secret manager не имитируются локальными «галочками»: это отдельный production-модуль с enrollment/recovery и инфраструктурой секретов. Они перечислены в `ROADMAP.md`.
