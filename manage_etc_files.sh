#!/bin/sh
set -x

# nginx
mkdir -p ~/etc/nginx
sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.org
sudo mv /etc/nginx/nginx.conf ~/etc/nginx/nginx.conf
sudo chmod 666 ~/etc/nginx/nginx.conf
sudo ln -s ~/etc/nginx/nginx.conf /etc/nginx/nginx.conf

# mysql
mkdir -p ~/etc/mysql
sudo cp /etc/mysql/mysql.conf.d/mysqld.cnf /etc/mysql/mysql.conf.d/mysqld.cnf.org
sudo cp /etc/mysql/mysql.conf.d/mysqld.cnf ~/etc/mysql/
sudo chmod 666 ~/etc/mysql/mysqld.cnf