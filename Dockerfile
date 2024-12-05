# Dockerfile
FROM python:3.11-slim

# Set the working directory in the container
WORKDIR /usr/src/app

# Install build dependencies needed for compiling packages like TgCrypto
RUN apt-get update && apt-get install -y \
    gcc \
    libc-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy the current directory contents into the container at /usr/src/app
COPY . .

# Upgrade pip and install dependencies
RUN pip install --upgrade pip
RUN pip install --no-cache-dir -r requirements.txt

# Make port 80 available to the world outside this container
EXPOSE 80

# Define environment variable
ENV NAME ArchiveBot

# Run the bot
CMD ["python", "./archive_bot.py"]