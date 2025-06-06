#!/bin/bash
# 日本語ロケールのインストールと設定
sudo apt-get update
sudo apt-get install -y --no-install-recommends locales language-pack-ja
sudo locale-gen ja_JP.UTF-8
sudo update-locale LANG=ja_JP.UTF-8

# プロファイルにロケール設定を追加
echo 'export LANG=ja_JP.UTF-8' | sudo tee -a /etc/profile.d/locale.sh
echo 'export LC_ALL=ja_JP.UTF-8' | sudo tee -a /etc/profile.d/locale.sh

# aquaのパスを各シェルの設定ファイルに追加
echo 'export PATH="/home/vscode/.local/share/aquaproj-aqua/bin:$PATH"' >> ~/.bashrc
echo 'export PATH="/home/vscode/.local/share/aquaproj-aqua/bin:$PATH"' >> ~/.zshrc
