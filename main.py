import telebot
import pandas as pd
import json
import os
from dotenv import load_dotenv
from datetime import datetime

# Load environment variables from .env file
load_dotenv()
# Replace 'YOUR_BOT_TOKEN' with the token you received from BotFather
BOT_TOKEN = os.getenv("BOT_TOKEN")
# Create a bot instance
bot = telebot.TeleBot(BOT_TOKEN)
# Read the CSV file
df = pd.read_csv("songbook.csv")
user_states = {}

def convert_datetimes(obj):
            if isinstance(obj, datetime):
                return obj.isoformat()
            elif isinstance(obj, dict):
                return {k: convert_datetimes(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                return [convert_datetimes(item) for item in obj]
            return obj

def create_user_info_json(message, user_state=None):
    """
    Create a comprehensive JSON with user information
    """
    user_info = {
        "personal_info": {
            "telegram_id": message.from_user.id,
            "username": message.from_user.username,
            "first_name": message.from_user.first_name,
            "last_name": message.from_user.last_name,
            "language_code": message.from_user.language_code,
        },
        "current_state": user_state if user_state else "no active interaction",
    }

    # Add song details if available in user state
    if user_state and isinstance(user_state, dict):
        user_info["song_selection"] = {
            "song_id": user_state.get("song_id", "N/A"),
            "song_name": user_state.get("song_name", "N/A"),
            "stage": user_state.get("stage", "N/A"),
        }

    return user_info


# Handler for /me command
@bot.message_handler(commands=["me"])
def show_user_info(message):
    try:
        # Check if user has an active state
        user_id = message.from_user.id
        user_state = user_states.get(user_id)
        # Create user info JSON
        user_info = create_user_info_json(message, user_state)
        # Apply the datetime conversion to the user_info dictionary
        user_info_copy = convert_datetimes(user_info)
        user_info_str = json.dumps(user_info_copy, ensure_ascii=False, indent=4)
        # Send user info as a message
        bot.reply_to(
            message,
            f"Your current user information:\n```json\n{user_info_str}\n```",
            parse_mode="Markdown",
        )
    except Exception as e:
            # Handle any potential errors
            bot.reply_to(message, f"Произошла ошибка: {str(e)}")


# Main message handler
@bot.message_handler(func=lambda message: True)
def echo_all(message):
    # Check if message starts with /start followed by an ID
    if message.text.startswith("/start "):
        try:
            # Extract the ID after /start
            id = message.text.split("/start ", 1)[1]
            # Find the specific row in the DataFrame
            specific_row = df.loc[df["ID"] == id]
            # Check if the row exists
            if not specific_row.empty:
                # Get the song name
                song_name = specific_row["песня"].values[0]
                # Store the song ID in user's state
                user_states[message.from_user.id] = {
                    "song_id": id,
                    "song_name": song_name,
                    "stage": "asking_name",
                }
                # Ask for the user's name
                bot.reply_to(
                    message, f"Привет! Песня {song_name} выбрана. Как тебя зовут?"
                )
            else:
                # ID not found
                bot.reply_to(message, "Извините, песня с таким ID не найдена.")
        except Exception as e:
            # Handle any potential errors
            bot.reply_to(message, f"Произошла ошибка: {str(e)}")
    # Handle user's name input
    elif (
        message.from_user.id in user_states
        and user_states[message.from_user.id]["stage"] == "asking_name"
    ):
        # Get the user's name
        user_name = message.text
        # Retrieve the song details from user's state
        song_id = user_states[message.from_user.id]["song_id"]
        song_name = user_states[message.from_user.id]["song_name"]
        # Update user state with name
        user_states[message.from_user.id]["user_name"] = user_name
        user_states[message.from_user.id]["time_added"] = datetime.now()
        user_states[message.from_user.id]["stage"] = "in_line"
        # Send a confirmation message
        bot.reply_to(
            message, f"привет, {user_name}! ты выбрал песню "{song_name}""
        )


# Start the bot
def main():
    print("Bot is running...")
    bot.infinity_polling()


if __name__ == "__main__":
    main()
