# TCPTUNNEL 基于 TCP 协议的隧道服务

    1. 在含有公网地址的计算机上运行`tunnel-server`启动隧道服务端
    2. 在内网计算机上启动`tunnel-client`并连接到公网隧道服务.
    3. 用户访问公网地址便可访问到内网的计算机.

## 程序基本信息

### 监听端口

    编译好的可执行文件在`bin`目录下.

| 程序          | 描述                                               | 端口                                      | 配置           |
| ------------- | -------------------------------------------------- | ----------------------------------------- | -------------- |
| tunnel-server | 隧道服务端, 提供隧道服务和用户访问服务             | 0.0.0.0:8080(用户端) 0.0.0.0:8101(隧道端) | 通过命令行指定 |
| tunnel-client | 隧道客户端, 在隧道服务端和被代理的目标机器间做转发 | 无                                        | 无             |

### 命令行清单

| 所属程序      | KEY       | 默认值         | 可选值        | 描述                                                                 |
| ------------- | --------- | -------------- | ------------- | -------------------------------------------------------------------- |
| tunnel-server | `listen`  | 0.0.0.0:8080   | `*`           | 用户访问地址, 用于接受用户端请求                                     |
| tunnel-server | `tunnel`  | 0.0.0.0:8101   | `*`           | 隧道通讯地址, 用户服务端和客户端通信                                 |
| tunnel-server | `speed`  | 0   | 整数           | 用于限制服务端数据转发速度, 默认'0'不限制, 单位: KB/S                                 |
| tunnel-server | `debug`   | false          | `true\|false` | 指定是否输出更多的调试日志                                           |
| tunnel-client | `tunnel`  | 127.0.0.1:8101 | `*`           | 隧道服务端地址, 连接服务端后才能正常使用                             |
| tunnel-client | `proxy`   | 127.0.0.1:80   | `*`           | 被代理的目标机器, 指定需要被访问的目标服务, 如: RDP, SSH, WEB 等服务 |
| tunnel-client | `debug`   | false          | `true\|false` | 指定是否输出更多的调试日志                                           |
| tunnel-client | `maxconn` | 25             | `*`           | 指定最大的空闲隧道个数, 不是越多越好                                 |

### 简单示例

假设公网 IP 为`101.133.123.123`, 内网机器`192.168.2.9`运行着 windows 系统, 现在需要通过公网`101.133.123.123`远程到内网`192.168.2.9`. 已知远程桌面(RDP)默认端口为`3389`.

1. 在公网(`101.133.123.123`)上开放`8101`和`3389`端口并启动`tunnel-server`服务

   `./tunnel-server --listen=0.0.0.0:3389`

2. 在内网机器(`192.168.2.9`)或可以访问到这台机器的机器上运行`tunnel-client`

   `./tunnel-client --tunnel=101.133.123.123:8101 --proxy=127.0.0.1:3389`

   或

   `./tunnel-client --tunnel=101.133.123.123:8101 --proxy=192.168.2.9:3389`

3. 使用远程桌面访问公网(`101.133.123.123`)即可

### 待办事项

1. 通信安全增强, 服务端客户端认证
2. 支持通过页面配置进行单程序多端口监听, 动态配置服务端和客户端，以及根据域名路由功能
