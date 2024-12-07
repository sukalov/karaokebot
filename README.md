# telegram bot for karaoke

simple python telegram bot, connected to [karaoke congbook](https://github.com/sukalov/karaoke). bot tracks the line for singing and songs

to run:
```bash
# create virtual environment
python3 -m venv .venv

# run it
source .venv/bin/activate

# install dependencies
pip install -r requirements.txt

# create .env
BOT_TOKEN=your_telegram_bot_token

# run
nodemon --exec python3 main.py
