#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}致命错误: ${plain} 请使用 root 权限运行此脚本\n" && exit 1

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    echo ""
    echo -e "${red}检查服务器操作系统失败，请联系作者!${plain}" >&2
    exit 1
fi
echo ""
echo -e "${green}---------->>>>>目前服务器的操作系统为: $release${plain}"

arch() {
    case "$(uname -m)" in
        x86_64 | x64 | amd64 ) echo 'amd64' ;;
        i*86 | x86 ) echo '386' ;;
        armv8* | armv8 | arm64 | aarch64 ) echo 'arm64' ;;
        armv7* | armv7 | arm ) echo 'armv7' ;;
        armv6* | armv6 ) echo 'armv6' ;;
        armv5* | armv5 ) echo 'armv5' ;;
        armv5* | armv5 ) echo 's390x' ;;
        *) echo -e "${green}不支持的CPU架构! ${plain}" && rm -f install.sh && exit 1 ;;
    esac
}

echo ""
echo -e "${yellow}---------->>>>>当前系统的架构为: $(arch)${plain}"
echo ""
last_version=$(curl -Ls "https://api.github.com/repos/xeefei/3x-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
# 获取 x-ui 版本
xui_version=$(/usr/local/x-ui/x-ui -v)

# 检查 xui_version 是否为空
if [[ -z "$xui_version" ]]; then
    echo ""
    echo -e "${red}------>>>当前服务器没有安装任何 x-ui 系列代理面板${plain}"
    echo ""
    echo -e "${green}-------->>>>片刻之后脚本将会自动引导安装〔3X-UI优化版〕${plain}"
else
    # 检查版本号中是否包含冒号
    if [[ "$xui_version" == *:* ]]; then
        echo -e "${green}---------->>>>>当前代理面板的版本为: ${red}其他 x-ui 分支版本${plain}"
        echo ""
        echo -e "${green}-------->>>>片刻之后脚本将会自动引导安装〔3X-UI优化版〕${plain}"
    else
        echo -e "${green}---------->>>>>当前代理面板的版本为: ${red}〔3X-UI优化版〕v${xui_version}${plain}"
    fi
fi
echo ""
echo -e "${yellow}---------------------->>>>>〔3X-UI优化版〕最新版为：${last_version}${plain}"
sleep 4

os_version=$(grep -i version_id /etc/os-release | cut -d \" -f2 | cut -d . -f1)

if [[ "${release}" == "arch" ]]; then
    echo "您的操作系统是 ArchLinux"
elif [[ "${release}" == "manjaro" ]]; then
    echo "您的操作系统是 Manjaro"
elif [[ "${release}" == "armbian" ]]; then
    echo "您的操作系统是 Armbian"
elif [[ "${release}" == "alpine" ]]; then
    echo "您的操作系统是 Alpine Linux"
elif [[ "${release}" == "opensuse-tumbleweed" ]]; then
    echo "您的操作系统是 OpenSUSE Tumbleweed"
elif [[ "${release}" == "centos" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        echo -e "${red} 请使用 CentOS 8 或更高版本 ${plain}\n" && exit 1
    fi
elif [[ "${release}" == "ubuntu" ]]; then
    if [[ ${os_version} -lt 20 ]]; then
        echo -e "${red} 请使用 Ubuntu 20 或更高版本!${plain}\n" && exit 1
    fi
elif [[ "${release}" == "fedora" ]]; then
    if [[ ${os_version} -lt 36 ]]; then
        echo -e "${red} 请使用 Fedora 36 或更高版本!${plain}\n" && exit 1
    fi
elif [[ "${release}" == "debian" ]]; then
    if [[ ${os_version} -lt 11 ]]; then
        echo -e "${red} 请使用 Debian 11 或更高版本 ${plain}\n" && exit 1
    fi
elif [[ "${release}" == "almalinux" ]]; then
    if [[ ${os_version} -lt 9 ]]; then
        echo -e "${red} 请使用 AlmaLinux 9 或更高版本 ${plain}\n" && exit 1
    fi
elif [[ "${release}" == "rocky" ]]; then
    if [[ ${os_version} -lt 9 ]]; then
        echo -e "${red} 请使用 RockyLinux 9 或更高版本 ${plain}\n" && exit 1
    fi
elif [[ "${release}" == "oracle" ]]; then
    if [[ ${os_version} -lt 8 ]]; then
        echo -e "${red} 请使用 Oracle Linux 8 或更高版本 ${plain}\n" && exit 1
    fi
else
    echo -e "${red}此脚本不支持您的操作系统。${plain}\n"
    echo "请确保您使用的是以下受支持的操作系统之一："
    echo "- Ubuntu 20.04+"
    echo "- Debian 11+"
    echo "- CentOS 8+"
    echo "- Fedora 36+"
    echo "- Arch Linux"
    echo "- Manjaro"
    echo "- Armbian"
    echo "- Alpine Linux"
    echo "- AlmaLinux 9+"
    echo "- Rocky Linux 9+"
    echo "- Oracle Linux 8+"
    echo "- OpenSUSE Tumbleweed"
    exit 1

fi

install_base() {
    case "${release}" in
    ubuntu | debian | armbian)
        apt-get update && apt-get install -y -q wget curl tar tzdata
        ;;
    centos | almalinux | rocky | oracle)
        yum -y update && yum install -y -q wget curl tar tzdata
        ;;
    fedora)
        dnf -y update && dnf install -y -q wget curl tar tzdata
        ;;
    arch | manjaro)
        pacman -Syu && pacman -Syu --noconfirm wget curl tar tzdata
        ;;
    alpine)
        apk update && apk add --no-cache wget curl tar tzdata
        ;;
    opensuse-tumbleweed)
        zypper refresh && zypper -q install -y wget curl tar timezone
        ;;
    *)
        apt-get update && apt install -y -q wget curl tar tzdata
        ;;
    esac
}

gen_random_string() {
    local length="$1"
    local random_string=$(LC_ALL=C tr -dc 'a-zA-Z0-9' </dev/urandom | fold -w "$length" | head -n 1)
    echo "$random_string"
}

# This function will be called when user installed x-ui out of security
config_after_install() {
    echo -e "${yellow}安装/更新完成！ 为了您的面板安全，建议修改面板设置 ${plain}"
    echo ""
    read -p "$(echo -e "${green}想继续修改吗？${red}选择“n”以保留旧设置${plain} [y/n]？--->>请输入：")" config_confirm
    if [[ "${config_confirm}" == "y" || "${config_confirm}" == "Y" ]]; then
        read -p "请设置您的用户名: " config_account
        echo -e "${yellow}您的用户名将是: ${config_account}${plain}"
        read -p "请设置您的密码: " config_password
        echo -e "${yellow}您的密码将是: ${config_password}${plain}"
        read -p "请设置面板端口: " config_port
        echo -e "${yellow}您的面板端口号为: ${config_port}${plain}"
        read -p "请设置面板登录访问路径（访问方式演示：ip:端口号/路径/）: " config_webBasePath
        echo -e "${yellow}您的面板访问路径为: ${config_webBasePath}${plain}"
        echo -e "${yellow}正在初始化，请稍候...${plain}"
        /usr/local/x-ui/x-ui setting -username ${config_account} -password ${config_password}
        echo -e "${yellow}用户名和密码设置成功!${plain}"
        /usr/local/x-ui/x-ui setting -port ${config_port}
        echo -e "${yellow}面板端口号设置成功!${plain}"
        /usr/local/x-ui/x-ui setting -webBasePath ${config_webBasePath}
        echo -e "${yellow}面板登录访问路径设置成功!${plain}"
        echo ""
    else
        echo ""
        sleep 1
        echo -e "${red}--------------->>>>Cancel...--------------->>>>>>>取消修改...${plain}"
        echo ""
        if [[ ! -f "/etc/x-ui/x-ui.db" ]]; then
            local usernameTemp=$(head -c 6 /dev/urandom | base64)
            local passwordTemp=$(head -c 6 /dev/urandom | base64)
            local webBasePathTemp=$(gen_random_string 10)
            /usr/local/x-ui/x-ui setting -username ${usernameTemp} -password ${passwordTemp} -webBasePath ${webBasePathTemp}
            echo -e "${yellow}检测到为全新安装，出于安全考虑将生成随机登录信息:${plain}"
            echo -e "###############################################"
            echo -e "${green}用户名: ${usernameTemp}${plain}"
            echo -e "${green}密  码: ${passwordTemp}${plain}"
            echo -e "${green}访问路径: ${webBasePathTemp}${plain}"
            echo -e "###############################################"
            echo -e "${green}如果您忘记了登录信息，可以在安装后通过 x-ui 命令然后输入${red}数字 10 选项${green}进行查看${plain}"
        else
            echo -e "${green}此次操作属于版本升级，保留之前旧设置项，登录方式保持不变${plain}"
            echo ""
            echo -e "${green}如果您忘记了登录信息，您可以通过 x-ui 命令然后输入${red}数字 10 选项${green}进行查看${plain}"
            echo ""
            echo ""
        fi
    fi
    sleep 1
    echo -e ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
    echo ""
    /usr/local/x-ui/x-ui migrate
}

echo ""
install_x-ui() {
    cd /usr/local/

    if [ $# == 0 ]; then
        last_version=$(curl -Ls "https://api.github.com/repos/xeefei/3x-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$last_version" ]]; then
            echo -e "${red}获取 3x-ui 版本失败，可能是 Github API 限制，请稍后再试${plain}"
            exit 1
        fi
        echo ""
        echo -e "-----------------------------------------------------"
        echo -e "${green}--------->>获取 3x-ui 最新版本：${yellow}${last_version}${plain}${green}，开始安装...${plain}"
        echo -e "-----------------------------------------------------"
        echo ""
        sleep 2
        echo -e "${green}---------------->>>>>>>>>安装进度50%${plain}"
        sleep 3
        echo ""
        echo -e "${green}---------------->>>>>>>>>>>>>>>>>>>>>安装进度100%${plain}"
        echo ""
        sleep 2
        wget -N --no-check-certificate -O /usr/local/x-ui-linux-$(arch).tar.gz https://github.com/xeefei/3x-ui/releases/download/${last_version}/x-ui-linux-$(arch).tar.gz
        if [[ $? -ne 0 ]]; then
            echo -e "${red}下载 3x-ui 失败, 请检查服务器是否可以连接至 GitHub？ ${plain}"
            exit 1
        fi
    else
        last_version=$1
        url="https://github.com/xeefei/3x-ui/releases/download/${last_version}/x-ui-linux-$(arch).tar.gz"
        echo ""
        echo -e "--------------------------------------------"
        echo -e "${green}---------------->>>>开始安装 3x-ui $1${plain}"
        echo -e "--------------------------------------------"
        echo ""
        sleep 2
        echo -e "${green}---------------->>>>>>>>>安装进度50%${plain}"
        sleep 3
        echo ""
        echo -e "${green}---------------->>>>>>>>>>>>>>>>>>>>>安装进度100%${plain}"
        echo ""
        sleep 2
        wget -N --no-check-certificate -O /usr/local/x-ui-linux-$(arch).tar.gz ${url}
        if [[ $? -ne 0 ]]; then
            echo -e "${red}下载 3x-ui $1 失败, 请检查此版本是否存在 ${plain}"
            exit 1
        fi
    fi

    if [[ -e /usr/local/x-ui/ ]]; then
        systemctl stop x-ui
        rm /usr/local/x-ui/ -rf
    fi
    
    sleep 3
    echo -e "${green}------->>>>>>>>>>>检查并保存安装目录${plain}"
    echo ""
    tar zxvf x-ui-linux-$(arch).tar.gz
    rm x-ui-linux-$(arch).tar.gz -f
    cd x-ui
    chmod +x x-ui

    # Check the system's architecture and rename the file accordingly
    if [[ $(arch) == "armv5" || $(arch) == "armv6" || $(arch) == "armv7" ]]; then
        mv bin/xray-linux-$(arch) bin/xray-linux-arm
        chmod +x bin/xray-linux-arm
    fi

    chmod +x x-ui bin/xray-linux-$(arch)
    cp -f x-ui.service /etc/systemd/system/
    wget --no-check-certificate -O /usr/bin/x-ui https://raw.githubusercontent.com/xeefei/3x-ui/main/x-ui.sh
    chmod +x /usr/local/x-ui/x-ui.sh
    chmod +x /usr/bin/x-ui
    sleep 2
    echo -e "${green}------->>>>>>>>>>>保存成功${plain}"
    sleep 2
    echo ""
    config_after_install

    systemctl daemon-reload
    systemctl enable x-ui
    systemctl start x-ui
    systemctl stop warp-go >/dev/null 2>&1
    wg-quick down wgcf >/dev/null 2>&1
    ipv4=$(curl -s4m8 ip.p3terx.com -k | sed -n 1p)
    ipv6=$(curl -s6m8 ip.p3terx.com -k | sed -n 1p)
    systemctl start warp-go >/dev/null 2>&1
    wg-quick up wgcf >/dev/null 2>&1

    echo ""
    echo -e "------->>>>${green}3x-ui ${last_version}${plain}<<<<安装成功，正在启动..."
    sleep 1
    echo ""
    echo -e "         ---------------------"
    echo -e "         |${green}3X-UI 控制菜单用法 ${plain}|${plain}"
    echo -e "         |  ${yellow}一个更好的面板   ${plain}|${plain}"   
    echo -e "         | ${yellow}基于Xray Core构建 ${plain}|${plain}"  
    echo -e "--------------------------------------------"
    echo -e "x-ui              - 进入管理脚本"
    echo -e "x-ui start        - 启动 3x-ui 面板"
    echo -e "x-ui stop         - 关闭 3x-ui 面板"
    echo -e "x-ui restart      - 重启 3x-ui 面板"
    echo -e "x-ui status       - 查看 3x-ui 状态"
    echo -e "x-ui settings     - 查看当前设置信息"
    echo -e "x-ui enable       - 启用 3x-ui 开机启动"
    echo -e "x-ui disable      - 禁用 3x-ui 开机启动"
    echo -e "x-ui log          - 查看 3x-ui 运行日志"
    echo -e "x-ui banlog       - 检查 Fail2ban 禁止日志"
    echo -e "x-ui update       - 更新 3x-ui 面板"
    echo -e "x-ui custom       - 自定义 3x-ui 版本"
    echo -e "x-ui install      - 安装 3x-ui 面板"
    echo -e "x-ui uninstall    - 卸载 3x-ui 面板"
    echo -e "--------------------------------------------"
    echo ""
    # if [[ -n $ipv4 ]]; then
    #    echo -e "${yellow}面板 IPv4 访问地址为：${green}http://$ipv4:${config_port}/${config_webBasePath}${plain}"
    # fi
    # if [[ -n $ipv6 ]]; then
    #    echo -e "${yellow}面板 IPv6 访问地址为：${green}http://[$ipv6]:${config_port}/${config_webBasePath}${plain}"
    # fi
    #    echo -e "请自行确保此端口没有被其他程序占用，${yellow}并且确保${red} ${config_port} ${yellow}端口已放行${plain}"
    sleep 3
    echo -e ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
    echo ""
    echo -e "${yellow}----->>>3X-UI面板和Xray启动成功<<<-----${plain}"
}
install_base
install_x-ui $1
echo ""
echo -e "----------------------------------------------"
sleep 4
info=$(/usr/local/x-ui/x-ui setting -show true)
echo -e "${info}${plain}"
echo ""
echo -e "若您忘记了上述面板信息，后期可通过x-ui命令进入脚本${red}输入数字〔10〕选项获取${plain}"
echo ""
echo -e "----------------------------------------------"
echo ""
sleep 2
echo -e "${green}安装/更新完成，若在使用过程中有任何问题${plain}"
echo -e "${yellow}请先描述清楚所遇问题加〔3X-UI〕中文交流群${plain}"
echo -e "${yellow}在TG群中${red} https://t.me/XUI_CN ${yellow}截图进行反馈${plain}"
echo ""
echo -e "----------------------------------------------"
echo ""
echo -e "${green}〔3X-UI〕优化版项目地址：${yellow}https://github.com/xeefei/3x-ui${plain}" 
echo ""
echo -e "${green} 详细安装教程：${yellow}https://xeefei.github.io/xufei/2024/05/3x-ui/${plain}"
echo ""
echo -e "----------------------------------------------"
echo ""
