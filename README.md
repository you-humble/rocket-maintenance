# Rocket Maintenance

![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/you-humble/e0883b0b3af0adeffb3b01446a0d4223/raw/coverage.json)


## CI/CD

Проект использует GitHub Actions для непрерывной интеграции и доставки. Основные workflow:

- **CI** (`.github/workflows/ci.yml`) - проверяет код при каждом push и pull request
  - Линтинг кода
  - Проверка безопасности
  - Выполняется автоматическое извлечение версий из Taskfile.yml
