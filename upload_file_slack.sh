#!/bin/bash

# 引数からファイル名とチャンネル名を取得
file_name=$1
channel_name=$2

# 環境変数からトークンを取得
token=$SLACK_TOKEN

# curlコマンドを実行
curl -F file=@"$file_name" -F filename="$file_name" -F channels="$channel_name" -H "Authorization: Bearer $token" https://slack.com/api/files.upload > /dev/null
