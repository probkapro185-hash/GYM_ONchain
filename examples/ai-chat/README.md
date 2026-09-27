# Demo AI chat

Это автономный frontend-пример для RAG API.

1. Запустите backend на `http://localhost:8080`.
2. Убедитесь, что `CORS_ALLOWED_ORIGIN` содержит `http://localhost:3000`.
3. В этой папке запустите:

```bash
python3 -m http.server 3000
```

4. Откройте `http://localhost:3000`.
5. Вставьте JWT пользователя после обычного `/api/v1/auth/login`.

Для production токен не нужно вводить вручную: используйте существующий auth-state вашего frontend.
