# lnx-voice-in-go

**Говорите — текст появляется там, где у вас курсор.** Этот проект превращает Linux-рабочий стол в точку, где речь быстро и **без облака** превращается в обычный набор символов: диктуйте в чат, документ, IDE или форму в браузере. Распознавание идёт **локально** на базе **whisper.cpp**, при желании с ускорением **CUDA**, так что голос не уходит на чужие серверы, а скорость зависит от вашего железа и выбранной модели. Сверху — лёгкое окно **Fyne**: уровень микрофона, старт/стоп записи и результат без лишнего «сказочного» интерфейса.

**Speak where you type — privately, on your machine.** A lean Linux utility that runs **Whisper** locally (optionally **GPU-accelerated**), shows a small overlay, and injects transcribed text into whatever window is focused—no subscription API, just mic, model, and keyboard injection.

## Интерфейс / What it looks like

Если вы просто открыли репозиторий и хотите понять, **что увидит пользователь на экране**: это не «полноэкранное приложение», а компактное **окно-оверлей** (Fyne) поверх других программ. На скриншоте ниже — типичный вид во время работы: индикатор уровня с микрофона, управление записью и область, куда попадает распознанный текст. Именно из этого окна текст затем **подставляется в активное поле ввода** (чат, документ, форма в браузере и т.д.) там, где у вас курсор.

**Browsing the repo?** The app is a small **desktop overlay**, not a full-screen program. The screenshot shows the usual layout while dictating: mic level, recording controls, and the transcription preview. From there, transcribed text is **injected into the focused input** wherever your cursor is (chat, editor, browser form, etc.).

![lnx-voice-in-go: оверлей с уровнем микрофона, записью и распознанным текстом / overlay with mic level, recording, and transcription](assets/images/form-screenshot.png)

---

## Документация / Documentation

| | Полное техническое описание |
|---|-----------------------------|
| **Русский** | [README.ru.md](README.ru.md) |
| **English** | [README.en.md](README.en.md) |

Там: стек, зависимости, сборка, модели, переменные окружения, лицензия.
