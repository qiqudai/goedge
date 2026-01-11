#!/bin/bash

set -o errexit

download(){
  local url1=$1
  local url2=$2
  local filename=$3

  speed1=`curl -m 5 -L -s -w '%{speed_download}' "$url1" -o /dev/null || true`
  speed1=${speed1%%.*}
  speed2=`curl -m 5 -L -s -w '%{speed_download}' "$url2" -o /dev/null || true`
  speed2=${speed2%%.*}
  echo "speed1:"$speed1
  echo "speed2:"$speed2
  url="$url1\n$url2"
  if [[ $speed2 -gt $speed1 ]]; then
    url="$url2\n$url1"
  fi
  echo -e $url | while read l;do
    echo "using url:"$l
    wget --dns-timeout=5 --connect-timeout=5 --read-timeout=10 --tries=2 "$l" -O $filename && break
  done
  
}

#判断系统版本
check_sys(){
    local checkType=$1
    local value=$2

    local release=''
    local systemPackage=''
    local packageSupport=''

    if [[ "$release" == "" ]] || [[ "$systemPackage" == "" ]] || [[ "$packageSupport" == "" ]];then

        if [[ -f /etc/redhat-release ]];then
            release="centos"
            systemPackage="yum"
            packageSupport=true

        elif cat /etc/issue | grep -q -E -i "debian";then
            release="debian"
            systemPackage="apt"
            packageSupport=true

        elif cat /etc/issue | grep -q -E -i "ubuntu";then
            release="ubuntu"
            systemPackage="apt"
            packageSupport=true

        elif cat /etc/issue | grep -q -E -i "centos|red hat|redhat";then
            release="centos"
            systemPackage="yum"
            packageSupport=true

        elif cat /proc/version | grep -q -E -i "debian";then
            release="debian"
            systemPackage="apt"
            packageSupport=true

        elif cat /proc/version | grep -q -E -i "ubuntu";then
            release="ubuntu"
            systemPackage="apt"
            packageSupport=true

        elif cat /proc/version | grep -q -E -i "centos|red hat|redhat";then
            release="centos"
            systemPackage="yum"
            packageSupport=true

        else
            release="other"
            systemPackage="other"
            packageSupport=false
        fi
    fi

    echo -e "release=$release\nsystemPackage=$systemPackage\npackageSupport=$packageSupport\n" > /tmp/ezhttp_sys_check_result

    if [[ $checkType == "sysRelease" ]]; then
        if [ "$value" == "$release" ];then
            return 0
        else
            return 1
        fi

    elif [[ $checkType == "packageManager" ]]; then
        if [ "$value" == "$systemPackage" ];then
            return 0
        else
            return 1
        fi

    elif [[ $checkType == "packageSupport" ]]; then
        if $packageSupport;then
            return 0
        else
            return 1
        fi
    fi
}

# 安装依赖
install_depend() {
    if check_sys sysRelease ubuntu;then
        apt-get -y install perl autoconf automake libtool libcurl3-dev libpcre3-dev libssl-dev zlib1g-dev g++ python-pip ipset python-dev
    elif check_sys sysRelease centos;then
        yum install -y perl autoconf automake libtool curl-devel pcre-devel openssl-devel zlib-devel gcc-c++   unzip ipset python-devel curl
        cd /etc/yum.repos.d/
        download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/epel.repo" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/epel.repo" "epel.repo"
        
        sed -i 's#https://#http://#g' /etc/yum.repos.d/epel*repo
        yum install -y python-pip || true
        if [[ `yum list installed  | grep python2-pip` == "" ]]; then
            sed -i 's#mirrors.aliyun.com#mirrors.tuna.tsinghua.edu.cn#' /etc/yum.repos.d/epel.repo
            yum install -y python-pip
        fi
    fi    
}

install_geoip() {
    mkdir -p /opt/geoip/
    cd /opt/geoip/
    download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/GeoLite2-Country.mmdb" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/GeoLite2-Country.mmdb" "GeoLite2-Country.mmdb"
}

install_python_module() {
    cd /tmp
    download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/pymodule-agent-20230102.tar.gz" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/pymodule-agent-20230102.tar.gz" "pymodule-agent-20230102.tar.gz"
    tar xf pymodule-agent-20230102.tar.gz
    cd pymodule-agent-20230102

    # 系统环境安装
    ## pip
    pip install pip-20.1.1-py2.py3-none-any.whl
    ## setuptools
    pip install setuptools-30.1.0-py2.py3-none-any.whl
    ## supervisor
    pip install supervisor-4.2.0-py2.py3-none-any.whl
    ## virtualenv
    pip install configparser-4.0.2-py2.py3-none-any.whl
    pip install scandir-1.10.0.tar.gz
    pip install typing-3.7.4.1-py2-none-any.whl
    pip install contextlib2-0.6.0.post1-py2.py3-none-any.whl
    pip install zipp-1.2.0-py2.py3-none-any.whl
    pip install six-1.15.0-py2.py3-none-any.whl
    pip install singledispatch-3.4.0.3-py2.py3-none-any.whl
    pip install distlib-0.3.0.zip
    pip install pathlib2-2.3.5-py2.py3-none-any.whl
    pip install importlib_metadata-1.6.1-py2.py3-none-any.whl
    pip install appdirs-1.4.4-py2.py3-none-any.whl
    pip install filelock-3.0.12.tar.gz
    pip install importlib_resources-2.0.1-py2.py3-none-any.whl
    pip install virtualenv-20.0.25-py2.py3-none-any.whl

    # 创建虚拟环境
    cd /opt
    python -m virtualenv -vv --extra-search-dir /tmp/pymodule-agent-20230102 --no-download --no-periodic-update venv
    ## 激活环境
    source /opt/venv/bin/activate

    # 虚拟环境安装
    cd /tmp/pymodule-agent-20230102

    ## Flask
    pip install click-7.1.2-py2.py3-none-any.whl
    pip install itsdangerous-1.1.0-py2.py3-none-any.whl
    pip install Werkzeug-1.0.1-py2.py3-none-any.whl 
    pip install MarkupSafe-1.1.1-cp27-cp27mu-manylinux1_x86_64.whl 
    pip install Jinja2-2.11.2-py2.py3-none-any.whl 
    pip install Flask-1.1.1-py2.py3-none-any.whl
    ## psutil
    #pip install psutil-5.7.0.tar.gz
    pip install psutil-5.8.0-cp27-cp27mu-manylinux2010_x86_64.whl
    ## bcrypt
    pip install pycparser-2.20-py2.py3-none-any.whl 
    pip install cffi-1.14.0-cp27-cp27mu-manylinux1_x86_64.whl 
    pip install six-1.15.0-py2.py3-none-any.whl 
    pip install bcrypt-3.1.7-cp27-cp27mu-manylinux1_x86_64.whl
    ## requests
    pip install certifi-2020.4.5.2-py2.py3-none-any.whl 
    pip install idna-2.9-py2.py3-none-any.whl
    pip install chardet-3.0.4-py2.py3-none-any.whl 
    pip install urllib3-1.25.9-py2.py3-none-any.whl
    pip install requests-2.24.0-py2.py3-none-any.whl
    ## requests_unixsocket
    pip install requests_unixsocket-0.2.0-py2.py3-none-any.whl
    ## pyOpenSSL
    pip install ipaddress-1.0.23-py2.py3-none-any.whl 
    pip install enum34-1.1.10-py2-none-any.whl 
    pip install cryptography-2.9.2-cp27-cp27mu-manylinux2010_x86_64.whl
    pip install pyOpenSSL-19.1.0-py2.py3-none-any.whl
    ## python_dateutil
    pip install python_dateutil-2.8.1-py2.py3-none-any.whl 
    ## APScheduler
    pip install funcsigs-1.0.2-py2.py3-none-any.whl 
    pip install futures-3.3.0-py2-none-any.whl 
    pip install pytz-2020.1-py2.py3-none-any.whl 
    pip install tzlocal-2.1-py2.py3-none-any.whl 
    pip install APScheduler-3.6.3-py2.py3-none-any.whl 
    ## gunicorn
    pip install gunicorn-19.10.0-py2.py3-none-any.whl
    ## gevent
    pip install zope.event-4.4-py2.py3-none-any.whl 
    pip install greenlet-2.0.1-cp27-cp27mu-manylinux2010_x86_64.whl
    pip install zope.interface-5.1.0-cp27-cp27mu-manylinux2010_x86_64.whl 
    pip install gevent-22.10.2-cp27-cp27mu-manylinux2010_x86_64.whl 
    ## requests_toolbelt
    pip install requests_toolbelt-0.9.1-py2.py3-none-any.whl 
    ## python_daemon
    pip install docutils-0.16-py2.py3-none-any.whl
    pip install lockfile-0.12.2-py2.py3-none-any.whl
    pip install python_daemon-2.2.4-py2.py3-none-any.whl

    ## redis
    pip install redis-3.5.3-py2.py3-none-any.whl

    ## Flask-Compress
    pip install Brotli-1.0.9-cp27-cp27mu-manylinux1_x86_64.whl
    pip install Flask-Compress-1.8.0.tar.gz

    # python_engineio
    pip install python_engineio-3.11.2-py2.py3-none-any.whl

    # python_socketio
    pip install python_socketio-4.4.0-py2.py3-none-any.whl

    # gevent-websocket
    pip install gevent-websocket-0.10.1.tar.gz

    # websocket_client
    pip install websocket_client-0.59.0-py2.py3-none-any.whl


    deactivate
}


install_openresty() {
    if [[ ! -d "/usr/local/openresty" ]]; then
        # openresty
        cd /usr/local
        download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/openresty-$SYS_VER-20230105.tar.gz" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/openresty-$SYS_VER-20230105.tar.gz" "openresty-$SYS_VER.tar.gz"
        tar xf openresty-$SYS_VER.tar.gz
        mkdir -p /data/nginx/cache
        mkdir -p /var/log/cdnfly/
        start_on_boot "ulimit -n 51200 && /usr/local/openresty/nginx/sbin/nginx"

        echo "/usr/local/openresty/libmaxminddb/lib/" > /etc/ld.so.conf.d/libmaxminddb.conf
        ldconfig

        # 下载spider_ip.json
        mkdir -p /usr/local/openresty/nginx/conf/vhost/
        cd /usr/local/openresty/nginx/conf/vhost/
        touch 0-9999999999-removing.conf
        download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/spider_ip.json" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/spider_ip.json" "spider_ip.json"
    fi
    
    # 下载rotate.tar.gz到/opt/cdnfly/nginx/conf，并解压
    cd /opt/cdnfly/nginx/conf
    download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/rotate.tar.gz" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/rotate.tar.gz" "rotate.tar.gz"
    tar xf rotate.tar.gz
    rm -f rotate.tar.gz

    chattr -R +i /usr/local/openresty/lualib || true
    chattr -R +i /usr/local/openresty/libmaxminddb > /dev/null 2>&1 || true
    chattr -R +i /usr/local/openresty/nginx/sbin/ || true
    chattr -R +i /usr/local/openresty/nginx/modules/ || true

    chattr +i /usr/local/openresty/nginx/conf || true
    chattr +i /usr/local/openresty/nginx/conf/listen_80.conf || true
    chattr +i /usr/local/openresty/nginx/conf/listen_other.conf || true
    chattr +i /usr/local/openresty/nginx/conf/nginx.conf || true

    chattr +i /usr/local/openresty/nginx/conf/stream > /dev/null 2>&1 || true
    chattr +i /usr/local/openresty/nginx/conf/stream/*.conf > /dev/null 2>&1 || true

    chattr +i /usr/local/openresty/nginx/conf/vhost || true
    chattr +i /usr/local/openresty/nginx/conf/vhost/*.conf || true

}

install_redis() {
    if [[ ! -d "/usr/local/redis" ]]; then
        cd /usr/local
        download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/redis-$SYS_VER-20200714.tar.gz" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/redis-$SYS_VER-20200714.tar.gz" "redis-$SYS_VER.tar.gz"
        tar xf redis-$SYS_VER.tar.gz
    fi
}

install_filebeat() {
    if [[ ! -d /etc/filebeat/ ]]; then
        if check_sys sysRelease ubuntu;then
            cd /tmp
            download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/filebeat-7.10.0-amd64.deb" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/filebeat-7.10.0-amd64.deb" "filebeat-7.10.0-amd64.deb"
            dpkg -i filebeat-7.10.0-amd64.deb

        elif check_sys sysRelease centos;then
            cd /tmp
            download "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/filebeat-7.10.0-x86_64.rpm" "https://raw.githubusercontent.com/Steady-WJ/cdnfly-kaixin/main/cdnfly/filebeat-7.10.0-x86_64.rpm" "filebeat-7.10.0-x86_64.rpm"
            yum install -y filebeat-7.10.0-x86_64.rpm
        fi

        mkdir -p /var/log/cdnfly/

    fi
    
    # 修改配置
    sed -i "s/192.168.0.30/$ES_IP/" /opt/cdnfly/agent/conf/filebeat.yml
    sed -i "s/ES_PWD/$ES_PWD/" /opt/cdnfly/agent/conf/filebeat.yml
    chmod 600 /opt/cdnfly/agent/conf/filebeat.yml

}

sync_time(){
    echo "start to sync time and add sync command to cronjob..."

    if [[ $ignore_ntp == false ]]; then
      if check_sys sysRelease ubuntu || check_sys sysRelease debian;then
          apt-get -y update
          apt-get -y install ntpdate wget
          /usr/sbin/ntpdate -u pool.ntp.org || true
          ! grep -q "/usr/sbin/ntpdate -u pool.ntp.org" /var/spool/cron/crontabs/root > /dev/null 2>&1 && echo '*/10 * * * * /usr/sbin/ntpdate -u pool.ntp.org > /dev/null 2>&1 || (date_str=`curl update.cdnfly.cn/common/datetime` && timedatectl set-ntp false && echo $date_str && timedatectl set-time "$date_str" )'  >> /var/spool/cron/crontabs/root
          service cron restart
      elif check_sys sysRelease centos; then
          yum -y install ntpdate wget
          /usr/sbin/ntpdate -u pool.ntp.org || true
          ! grep -q "/usr/sbin/ntpdate -u pool.ntp.org" /var/spool/cron/root > /dev/null 2>&1 && echo '*/10 * * * * /usr/sbin/ntpdate -u pool.ntp.org > /dev/null 2>&1 || (date_str=`curl update.cdnfly.cn/common/datetime` && timedatectl set-ntp false && echo $date_str && timedatectl set-time "$date_str" )' >> /var/spool/cron/root
          service crond restart
      fi
    fi

    # 时区
    rm -f /etc/localtime
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

    if /sbin/hwclock -w;then
        return
    fi 


}


config() {
    sed -i "s/127.0.0.1/$MASTER_IP/" /opt/cdnfly/agent/conf/config.py
    sed -i "s/192.168.0.30/$ES_IP/" /opt/cdnfly/agent/conf/config.py
    sed -i "s/ES_PWD =.*/ES_PWD = \"$ES_PWD\"/" /opt/cdnfly/agent/conf/config.py
    sed -i "s/MASTER_HOST =.*/MASTER_HOST = \"$MASTER_HOST\"/" /opt/cdnfly/agent/conf/config.py
}

start_on_boot(){
    local cmd="$1"
    if [[ -f "/etc/rc.local" ]]; then
        sed -i '/exit 0/d' /etc/rc.local
        if [[ `grep "${cmd}" /etc/rc.local` == "" ]];then 
            echo "${cmd}" >> /etc/rc.local
        fi 
        chmod +x /etc/rc.local
    fi


    if [[ -f "/etc/rc.d/rc.local" ]]; then
        sed -i '/exit 0/d' /etc/rc.d/rc.local
        if [[ `grep "${cmd}" /etc/rc.d/rc.local` == "" ]];then 
            echo "${cmd}" >> /etc/rc.d/rc.local
        fi 
        chmod +x /etc/rc.d/rc.local 
    fi 
}

install_rsyslog() {
# rsyslog
cat > /etc/rsyslog.d/cdnfly.conf <<'EOF'

    $ModLoad imudp
    $UDPServerRun 514
    $Umask 0000
    :msg,contains,"[cdnfly" /var/log/cdnfly.log
    $Umask 0022
    $EscapeControlCharactersOnReceive off
EOF

service rsyslog restart || true

mkdir -p /var/log/cdnfly/    

}

start() {
    start_on_boot "supervisord -c /opt/cdnfly/agent/conf/supervisord.conf"
    if ! supervisord -c /opt/cdnfly/agent/conf/supervisord.conf > /dev/null 2>&1;then
        supervisorctl -c /opt/cdnfly/agent/conf/supervisord.conf reload
    fi
    
    rm -rf /opt/cdnfly/master
    chmod +x /opt/cdnfly/agent/sh/*.sh

    # 关闭防火墙
    if check_sys sysRelease centos; then
        systemctl stop firewalld.service || true
        systemctl disable firewalld.service || true
    fi

    # 添加cdnfly ipset
    if ! ipset list cdnfly > /dev/null 2>&1; then
        ipset -N cdnfly iphash maxelem 10000000 timeout 3600
    fi

    if ! ipset list cdnfly_white > /dev/null 2>&1; then
        ipset -N cdnfly_white iphash maxelem 10000000 timeout 0
    fi

    if ! ipset list cdnfly_black > /dev/null 2>&1; then
        ipset -N cdnfly_black iphash maxelem 10000000 timeout 0
    fi

    # 添加iptables
    if [[ $(iptables -t filter -S INPUT 1 | grep -- '-A INPUT -m set --match-set cdnfly_white src -j ACCEPT') == "" ]];then
        iptables -D INPUT -m set --match-set cdnfly src -j DROP || true
        iptables -D INPUT -m set --match-set cdnfly_black src -j DROP || true
        iptables -D INPUT -m set --match-set cdnfly_white src -j ACCEPT || true
        

        iptables -I INPUT -m set --match-set cdnfly src -j DROP || true
        iptables -I INPUT -m set --match-set cdnfly_black src -j DROP || true
        iptables -I INPUT -m set --match-set cdnfly_white src -j ACCEPT || true
    fi

    # 添加cdnfly ipset ipv6
    if ! ipset list cdnfly_v6 > /dev/null 2>&1; then
        ipset create cdnfly_v6 hash:net family inet6 maxelem 10000000 timeout 3600
    fi

    if ! ipset list cdnfly_white_v6 > /dev/null 2>&1; then
        ipset create cdnfly_white_v6 hash:net family inet6 maxelem 10000000 timeout 0
    fi

    if ! ipset list cdnfly_black_v6 > /dev/null 2>&1; then
        ipset create cdnfly_black_v6 hash:net family inet6 maxelem 10000000 timeout 0
    fi

    # 添加iptables v6
    if [[ $(ip6tables -t filter -S INPUT 1 | grep -- '-A INPUT -m set --match-set cdnfly_white_v6 src -j ACCEPT') == "" ]];then
        ip6tables -D INPUT -m set --match-set cdnfly_v6 src -j DROP || true
        ip6tables -D INPUT -m set --match-set cdnfly_black_v6 src -j DROP || true
        ip6tables -D INPUT -m set --match-set cdnfly_white_v6 src -j ACCEPT || true
        

        ip6tables -I INPUT -m set --match-set cdnfly_v6 src -j DROP || true
        ip6tables -I INPUT -m set --match-set cdnfly_black_v6 src -j DROP || true
        ip6tables -I INPUT -m set --match-set cdnfly_white_v6 src -j ACCEPT || true
    fi
        
    ulimit -n 51200 && /usr/local/openresty/nginx/sbin/nginx || true
    echo "安装节点成功！"
    echo "开始往主控发送请求注册节点..."
    cmd="curl -kL --resolve $MASTER_HOST:88:$MASTER_IP --resolve $MASTER_HOST:443:$MASTER_IP  -X POST -H \"Host: $MASTER_HOST\"  http://$MASTER_HOST:88/new-node -H \"es-pwd: $ES_PWD\" "
    ret=`eval $cmd || true`
    if [[ "$ret" =~ "pending_node" || "$ret" =~ "success" ]];then
        echo "注册成功，请登录后台，切换到待初始化菜单, 来初始化节点"

    elif [[ "$ret" =~ "in node" ]];then
        node_id=`echo $ret | grep -Po "id:\d+" | sed 's/id://g'`
        sed -i "s/NODE_ID.*/NODE_ID = \"$node_id\"/" /opt/cdnfly/agent/conf/config.py
        supervisorctl -c /opt/cdnfly/agent/conf/supervisord.conf restart agent
        supervisorctl -c /opt/cdnfly/agent/conf/supervisord.conf restart task        
        echo "该节点已存在于CDN后台，请登录后台禁用并启动此节点完成初始化"

    else
        echo "注册失败，请手动执行如下命令来注册"
        echo $cmd
    fi

}

need_sys() {
    SYS_VER=`python -c "import platform;import re;sys_ver = platform.platform();sys_ver = re.sub(r'.*-with-(.*)-.*','\g<1>',sys_ver);print sys_ver;"`
    if [[ $SYS_VER =~ "Ubuntu-16.04" ]];then
      echo "$sys_ver"
    elif [[ $SYS_VER =~ "centos-7" ]]; then
      SYS_VER="centos-7"
      echo $SYS_VER
    else  
      echo "目前只支持ubuntu-16.04和Centos-7"
      exit 1
    fi
}

# 检查系统
need_sys

# 解析命令行参数
TEMP=`getopt -o h --long help,master-ip:,master-host:,es-ip:,es-pwd:,ignore-ntp -- "$@"`
if [ $? != 0 ] ; then echo "Terminating..." >&2 ; exit 1 ; fi
eval set -- "$TEMP"

ignore_ntp=false
while true ; do
    case "$1" in
        -h|--help) help ; exit 1 ;;
        --master-ip) MASTER_IP=$2 ; shift 2 ;;
        --master-host) MASTER_HOST=$2 ; shift 2 ;;
        --es-ip) ES_IP=$2 ; shift 2 ;;
        --es-pwd) ES_PWD=$2 ; shift 2 ;;
        --ignore-ntp) ignore_ntp=true; shift 1;;
        --) shift ; break ;;
        *) echo "Internal error!" ; exit 1 ;;
    esac
done

if [[ "$MASTER_IP" == "" ]]; then
    echo "please specify master ip with --master-ip 1.1.1.1 "
    exit 1
fi

if [[ "$ES_IP" == "" ]]; then
    echo "please specify elasticsearch ip with --es-ip 1.1.1.1 "
    exit 1
fi

if [[ "$ES_PWD" == "" ]]; then
    echo "please specify elasticsearch password with --es-pwd xxx "
    exit 1
fi

# 禁止安装节点到主控
if [[ -d "/opt/cdnfly/master" ]];then 
    echo "存在/opt/cdnfly/master，当前的机器可能是主控，请勿安装节点程序"
    exit 1
fi

sync_time
install_depend
install_geoip
install_python_module
install_openresty
install_redis
install_filebeat
install_rsyslog
config
start

