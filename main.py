import telebot
import pandas as pd
import json
import pytz
import threading
import os
from dotenv import load_dotenv
from datetime import datetime, timedelta

load_dotenv()
BOT_TOKEN = os.getenv("BOT_TOKEN")
ADMIN_BOT_TOKEN = os.getenv("ADMIN_BOT_TOKEN")

# create bot instances
bot = telebot.TeleBot(BOT_TOKEN)
admin = telebot.TeleBot(ADMIN_BOT_TOKEN)

# read the CSV file
df = pd.read_csv("songbook.csv")
user_states = {}

admins = ["sukalov", "olakotr", "yatskovanatoly", "motygapishet"]


def convert_datetimes(obj):
    moscow_tz = pytz.timezone("Europe/Moscow")

    def format_datetime(dt):
        # Ensure the datetime is timezone-aware
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=pytz.UTC)

        # Convert to Moscow time and format as a readable string
        moscow_time = dt.astimezone(moscow_tz)
        return moscow_time.strftime("%H:%M:%S")

    if isinstance(obj, datetime):
        return format_datetime(obj)
    elif isinstance(obj, dict):
        return {k: convert_datetimes(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [convert_datetimes(item) for item in obj]
    return obj


# Handler for /users command in user bot
@bot.message_handler(commands=["users"])
def show_users_info(message):
    try:
        # Apply the datetime conversion to the user_info dictionary
        user_info_copy = convert_datetimes(user_states)
        print(user_info_copy)
        user_info_str = json.dumps(user_info_copy, ensure_ascii=False, indent=4)
        # Send user info as a message
        bot.reply_to(
            message,
            f"```json\n{user_info_str}\n```",
            parse_mode="markdown",
        )
    except Exception as e:
        # Handle any potential errors
        bot.send_message(message.from_user.id, f"произошла ошибка: {str(e)}")


@bot.message_handler(commands=["exit"])
def exit_from_line(message):
    try:
        if message.from_user.id in user_states:
            if user_states[message.from_user.id]['stage'] == "in_line":
                bot.send_message(message.from_user.id, "ок, вычёркиваем")
            else:
                bot.send_message(message.from_user.id, "ок, не будем записывать")

            removed_song = user_states[message.from_user.id]["song_name"]
            removed_name = user_states[message.from_user.id]["username"]
            del user_states[message.from_user.id]

            # Optional: Send a message to the admin bot about the exit
            try:
                admin.send_message(
                    "sukalov",  # You'll need to define this
                    f"{removed_name} вышел из очереди. песня: {removed_song}",
                )
            except Exception as admin_notify_error:
                print(f"сould not notify admin: {admin_notify_error}")

        else:
            # User is not in the line
            bot.send_message(message.from_user.id, "вас и так нет в очереди..")

    except Exception as e:
        # Handle any potential errors
        bot.send_message(
            message.from_user.id, f"ошибка при выходе из очереди: {str(e)}"
        )

@bot.message_handler(commands=["line"])
def show_line(message):
    try:
        # Filter only users in the line
        line_users = {
            user_id: user_info
            for user_id, user_info in user_states.items()
            if user_info.get("stage") == "in_line"
        }

        # If no users in line
        if not line_users:
            bot.send_message(message.from_user.id, "в очереди никого нет")
            return

        # Create a formatted message with users in line
        line_message = "очередь:\n\n"
        for idx, (user_id, user_info) in enumerate(line_users.items(), 1):
            line_message += (
                f"{idx}. {user_info.get('typed_name', 'без имени')}\n"
                f"   песня: [{user_info.get('song_name', 'не указана')}]({user_info.get('song_link', 'link not found')})\n"
                f"   добавлен: {convert_datetimes(user_info.get('time_added'))}\n"
                f"   юзернейм: @{user_info.get('username', 'не указан')}\n\n"
            )

        # Send the formatted line information
        bot.send_message(
            message.from_user.id,
            line_message,
            parse_mode="Markdown",
            disable_web_page_preview=True,
        )

    except Exception as e:
        # Handle any potential errors
        bot.send_message(message.from_user.id, f"произошла ошибка: {str(e)}")

# Main message handler
@bot.message_handler(func=lambda message: True)
def main_handler(message):
    # Check if message starts with /start followed by an id
    if message.text.startswith("/start "):
        try:
            # Extract the ID after /start
            id = message.text.split("/start ", 1)[1]
            # Find the specific row in the DataFrame
            specific_row = df.loc[df["id"] == id]
            if specific_row["имя"].values.size == 0:
                name = specific_row["имя"].values[0]
            else:
                name = ""
            # Check if the row exists
            if not specific_row.empty:
                # Get the song name
                song_name = f"{name} {specific_row['исполнитель'].values[0]} - {specific_row['песня'].values[0]}".strip()
                song_link = specific_row["ссылка"].values[0]
                # Store the song ID in user's state
                user_states[message.from_user.id] = {
                    "username": message.from_user.username,
                    "tg_name": f"{message.from_user.first_name or ''} {message.from_user.last_name or ''}".strip(),
                    "song_id": id,
                    "song_name": song_name,
                    "song_link": song_link,
                    "stage": "asking_name",
                }
                # Ask for the user's name
                bot.send_message(
                    message.from_user.id,
                    f'привет! выбрана песня "{song_name}". чтобы записаться, скажите, как вас зовут зовут? (или того кто будет петь)\n\n'
                    f"если вы передумали — можете выписаться командой /exit",
                )
            else:
                # id not found
                bot.reply_to(message, "извините, песня с таким id не найдена")
        except Exception as e:
            # Handle any potential errors
            bot.send_message(message.from_user.id, f"произошла ошибка: {str(e)}")
    elif message.text.startswith("/start"):
        bot.send_message(
            message.from_user.id,
            "не, просто так не работает. выбор песен в сонгбуке: https://karaoke-songbook.netlify.app",
        )
    # Handle user's name input
    elif (
        message.from_user.id in user_states
        and user_states[message.from_user.id]["stage"] == "asking_name"
    ):
        # Get the user's name
        typed_name = message.text
        # Update user state with name
        user_states[message.from_user.id]["typed_name"] = typed_name
        # Retrieve the song details from user's state
        song_name = user_states[message.from_user.id]["song_name"]
        song_link = user_states[message.from_user.id]["song_link"]
        user_states[message.from_user.id]["time_added"] = datetime.now(pytz.UTC)
        user_states[message.from_user.id]["stage"] = "in_line"
        # Send a confirmation message
        bot.reply_to(
            message,
            f'отлично, {typed_name}! вы выбрали песню "{song_name}". скоро вас позовут на сцену\n\nа слова можно найти [здесь]({song_link})',
            parse_mode="Markdown",
        ),


clear_being_confirmed = False

@admin.message_handler(commands=["line"])
def show_line(message):
    try:
        # Filter only users in the line
        line_users = {
            user_id: user_info
            for user_id, user_info in user_states.items()
            if user_info.get("stage") == "in_line"
        }

        # If no users in line
        if not line_users:
            admin.send_message(message.from_user.id, "в очереди никого нет")
            return

        # Create a formatted message with users in line
        line_message = "очередь:\n\n"
        for idx, (user_id, user_info) in enumerate(line_users.items(), 1):
            line_message += (
                f"{idx}. {user_info.get('typed_name', 'без имени')}\n"
                f"   песня: [{user_info.get('song_name', 'не указана')}]({user_info.get('song_link', 'link not found')})\n"
                f"   добавлен: {convert_datetimes(user_info.get('time_added'))}\n"
                f"   юзернейм: @{user_info.get('username', 'не указан')}\n\n"
            )

        # Send the formatted line information
        admin.send_message(
            message.from_user.id,
            line_message,
            parse_mode="Markdown",
            disable_web_page_preview=True,
        )

    except Exception as e:
        # Handle any potential errors
        admin.send_message(message.from_user.id, f"произошла ошибка: {str(e)}")


@admin.message_handler(commands=["clear_line"])
def clear_line(message):
    global clear_being_confirmed
    if message.from_user.username in admins:
        try:
            # Send the formatted line information
            clear_being_confirmed = True
            admin.send_message(
                message.from_user.id,
                "вы хотите очистить список?\n  /yes                               /no",
            )

        except Exception as e:
            # Handle any potential errors
            admin.send_message(message.from_user.id, f"произошла ошибка: {str(e)}")
    else:
        admin.send_message(message.from_user.id, "вы не админ")


@admin.message_handler(commands=["yes"])
def confirm(message):
    global clear_being_confirmed
    global user_states

    if message.from_user.username in admins:
        if clear_being_confirmed == True:
            user_states = {}
            admin.send_message(message.from_user.id, "список очищен")
            clear_being_confirmed = False
        else:
            admin.send_message(message.from_user.id, "что yes?")
    else:
        admin.send_message(message.from_user.id, "вы не админ")


@admin.message_handler(commands=["no"])
def confirm(message):
    global clear_being_confirmed

    if message.from_user.username in admins:
        if clear_being_confirmed == True:
            admin.send_message(message.from_user.id, "ок, отмена")
            clear_being_confirmed = False
        else:
            admin.send_message(message.from_user.id, "что no?")
    else:
        admin.send_message(message.from_user.id, "вы не админ")


# Start the bot
def main():
    print("both karaoke bot and admin bot are running...")
    # Create threads for both polling functions
    bot_thread = threading.Thread(target=bot.infinity_polling)
    admin_thread = threading.Thread(target=admin.infinity_polling)
    # Start the threads
    bot_thread.start()
    admin_thread.start()


if __name__ == "__main__":
    main()
