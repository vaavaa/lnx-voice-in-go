# lnx-voice-in-go

**Говорите — текст появляется там, где у вас курсор.** Этот проект превращает Linux-рабочий стол в точку, где речь быстро и **без облака** превращается в обычный набор символов: диктуйте в чат, документ, IDE или форму в браузере. Распознавание идёт **локально** на базе **whisper.cpp**, при желании с ускорением **CUDA**, так что голос не уходит на чужие серверы, а скорость зависит от вашего железа и выбранной модели. Сверху — лёгкое окно **Fyne**: уровень микрофона, старт/стоп записи и результат без лишнего «сказочного» интерфейса.

**Speak where you type — privately, on your machine.** A lean Linux utility that runs **Whisper** locally (optionally **GPU-accelerated**), shows a small overlay, and injects transcribed text into whatever window is focused—no subscription API, just mic, model, and keyboard injection.

---

## Документация / Documentation

| | Полное техническое описание |
|---|-----------------------------|
| **Русский** | [README.ru.md](README.ru.md) |
| **English** | [README.en.md](README.en.md) |

Там: стек, зависимости, сборка, модели, переменные окружения, лицензия.
