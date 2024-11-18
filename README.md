# DataSoup

A feed of changes from data.gov.il (Open Goverment Data)

Currently publishes to telegram channel: [DataSoup](https://t.me/DataSoup)

## How to run

Create a file name `.telegram_token` with the telegram bot token.

Find and replace the `chat_id` in `main.go` with the chat id of the channel you want to publish to.

Build the project by running `go build`

Bootstrap the data(goes back one week) by running `main -bootstrap` (Takes up about 5.5 GB of disk space)

Run the project by running `main`

Every time you run the project it will go through[1] any changes made since last time, diff them and publish them to the telegram channel.

1. Currently its artificially **significantly** slowed down and runs sequentially to ease debuggging and avoid telegram api rate limits.

## What are Resources Datasets and Organizations?

**Organizations** are the entities that publish the data. E.g. Ministry of Health, Ministry of Education, etc.

**Datasets** is the logical grouping of data. E.g. Covid-19 data, School data, etc.

**Resources** consist of the actual data files. E.g. CSV files, XLSX files, etc. (We only care about CSV files for now)

**Tags** group Datasets by a common theme. E.g. Covid-19, Education, etc. (Not implemented yet)
