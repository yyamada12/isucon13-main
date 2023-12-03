#!/bin/sh
set -x

# nginx
mkdir -p ~/etc/nginx
sudo cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.org
sudo cp /etc/nginx/nginx.conf ~/etc/nginx/nginx.conf
sudo chmod a+w ~/etc/nginx/nginx.conf
# sudo ln -s ~/etc/nginx/nginx.conf /etc/nginx/nginx.conf

mkdir -p ~/etc/nginx/sites-enabled
sudo cp -r /etc/nginx/sites-enabled /etc/nginx/sites-enabled.org
sudo cp -r /etc/nginx/sites-enabled/* ~/etc/nginx/sites-enabled/
sudo chmod a+w ~/etc/nginx/sites-enabled/*


# mysql
mkdir -p ~/etc/mysql
sudo cp /etc/mysql/mysql.conf.d/mysqld.cnf /etc/mysql/mysql.conf.d/mysqld.cnf.org
sudo cp /etc/mysql/mysql.conf.d/mysqld.cnf ~/etc/mysql/
sudo chmod a+w ~/etc/mysql/mysqld.cnf

## ulimit
mkdir -p ~/etc/security
sudo cp /etc/security/limits.conf /etc/security/limits.conf.org
sudo cp /etc/security/limits.conf ~/etc/security/limits.conf
sudo chmod a+w ~/etc/security/limits.conf
