#add-apt-repository ppa:ubuntu-lxc/lxd-stable -y
#got error:
#fix: 
#http://lgogua.blogspot.com/2014/03/kali-linux-fix-problem-add-apt.html

wget http://blog.anantshri.info/content/uploads/2010/09/add-apt-repository.sh.txt
cp add-apt-repository.sh.txt /usr/sbin/add-apt-repository
chmod o+x /usr/sbin/add-apt-repository
chown root:root /usr/sbin/add-apt-repository
add-apt-repository ppa:ubuntu-lxc/lxd-stable -y

apt-get update
#ok

#apt-get -y install linux-image-extra-$(uname -r)
#skip

#apt-get -y install linux-image-extra-virtual
#skip

apt-get -y install curl
#ok

apt-get -y install git
#ok

apt-get -y install docker.io
#ok

#apt-get -y install docker-engine
#install docker-ce instead
apt-get -y install docker-ce

apt-get -y install cgroup-tools cgroup-bin
#ok

apt-get -y install python-pip
#ok

apt-get -y install python2.7-dev
#ok

pip install netifaces
#ok

pip install rethinkdb
#ok

pip install tornado
#ok

#INSTALLING GO:
wget http://dave.cheney.net/paste/go-linux-arm-bootstrap-c788a8e.tbz
tar xvjf go-linux-arm-bootstrap-c788a8e.tbz
su
mv go-linux-arm-bootstrap /root
mv go-linux-arm-bootstrap go1.4
exit

wget -q -O /tmp/go1.11.4.linux-arm64.tar.gz https://dl.google.com/go/go1.11.4.linux-arm64.tar.gz
#wget -q -O /tmp/go1.11.4.linux-amd64.tar.gz https://dl.google.com/go/go1.11.4.linux-amd64.tar.gz
#instead run wget -q -O /tmp/go1.11.4.linux-arm64.tar.gz https://dl.google.com/go/go1.11.4.linux-arm64.tar.gz

tar -C /usr/local -xzf /tmp/go1.11.4.linux-arm64.tar.gz
#tar -C /usr/local -xzf /tmp/go1.11.4.linux-amd64.tar.gz
#instead run tar -C /usr/local -xzf /tmp/go1.11.4.linux-arm64.tar.gz

cd /usr/local/go/src
sudo ./make.bash

ln -s /usr/local/go/bin/go /usr/bin/go

service docker restart
