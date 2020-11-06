# Auth
## 概览
* 访问控制功能有：用户组、用户、角色、访问白名单。而权限则是针对不同的角色以及用户组对用户数据过滤以及菜单可视控制。
* 权限控制力度：数据过滤、操作控制（增、删、改、查）、菜单可视。
* 只有超级管理员才拥有访问控制权限，普通用户只能修改自己的密码。
* 用户组、角色、用户之间的关系：多对多，即一个用户可以选择多个用户组、多个角色。一个角色也可以选择多个用户多个角色。用户所拥有的权限是选择的用户组以及角色的并集。

## 设计
#### 配置表 (ddi-role.json)
* 新版权限控制采用角色配置表的方式进行控制。配置文件在项目的：etc/ddi-role.json下面。
* 目前支持两种角色：（超级管理员）SUPER、（普通管理员）NORMAL。
    * SUPER默认为空表示支持所有权限。
    * (普通管理员）NORMAL:访问控制权限、系统管理模块所有权限、DNS递归安全无权限查看。在未分配DNS或者IP前缀之前其余的所有权限均为只读。
* 权限配置分为三大模块：（基础权限）baseAuthority、（DNS模块权限）dnsAuthority、（地址管理权限）dhcpAuthority。
* 主要控制参数资源（resource）、操作权限（operations）、DNS权限（views）、IP地址前缀（plans）、过滤开关（filter）。
* 资源（resource）：列表中出现的资源表示可见。目前普通用户可见的资源有：
    * （基础权限）baseAuthority:ddiuser（用户信息）、node（节点信息）、dns（DNS统计信息）、dhcp（DHCP统计信息）。
    * （DNS权限）dnsAuthority:dnsglobalconfig（DNS全局配置）、acl（acl访问控制列表）、view（视图列表）、zone（权威区）、rr（资源记录）、redirection（重定向）、urlredirect（url重定向）、forward（转发规则）、forwardzone（转发组）
    * （地址管理权限）dhcpAuthority:dhcpconfig（DHCP基础配置）、subnet（地址池管理）、pdpool（前缀委派）、pool（动态地址池）、reservation（固定地址）、staticaddress（静态地址）、plan（IP地址规划）、layout（IP地址规划面板）、networkinterface（IP扫描）、asset（终端管理）、networkequipment（设备管理）、networktopology(网络拓扑)、scannedsubnet（IP地址检测）、clientclass(option60)、netnode(网络节点)
* 操作权限（operations）:支持的类型有["GET","ACTION","POST","PUT","DELETE"]。除了ddiuser资源拥有["GET","ACTION","POST"]操作权限（用于读取用户信息以及修改用户密码）以外，其余的资源未授权下默认为["GET"]，即只读权限。
* DNS权限（views）:通过配置views列表搭配filter开关，来过滤dns的数据。例如给某个用户视图v1的操作权限，则dnsAuthority需要配为:
    ```
    { "resource": "view", "views":[v1], "operations": ["GET"] },
    { "resource": "zone", "views":[v1], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "rr", "views":[v1], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "redirection", "views":[v1], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "urlredirect", "views":[v1], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "forwardzone", "views":[v1], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] }
    ```
并且其关联的zone、rr、redirection、urlredirect、forwardzone都会加入v1视图的权限，并且这些资源的操作权限动态的便成为["GET","ACTION","POST","PUT","DELETE"],这样该用户就拥有视图v1下的所有操作权限，但视图view、acl、forward只有只读权限。
* IP地址前缀（plans）:该列表用于地址管理权限控制，根据IP地址规划获取前缀进行访问控制。只有赋予用户特定的前缀才能获取到该前缀下的数据以及操作。未分配IP前缀是看不到任何数据。
    1. 管理员进行IP地址规划，创建plan为2008::/60。通过访问控制给用户添加2008::/60权限。
    2. 用户的dhcpAuthority模块的权限变动为：
    ```
    { "resource": "plan", "plans":[2008::/60], "operations": ["GET"] },
    { "resource": "subnet", "plans":[2008::/60], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "asset", "plans":[2008::/60], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "networkequipment", "plans":[2008::/60], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] },
    { "resource": "scannedsubnet", "plans":[2008::/60], "filter": true, "operations": ["GET","ACTION","POST","PUT","DELETE"] }
    ```
同样用户对于plan只有只读权限，不能增删改，除了以上给出的资源其余的资源均只有只读权限。
* 数据过滤开关（filter）：主要用于控制操作权限变化，例如原本只有["GET"]，如果赋予该资源views或者plans以后如果filter为true，那么该资源的operations操作变为：["GET","ACTION","POST","PUT","DELETE"]。
* 权限控制流程：
    1. 每个用户身上维护一个RoleAuthority列表，其内容就是ddi-role.json加载进来的。创建或更新用户、用户组、角色都会更新这个表的内容。
    2. 首先判断客户端请求的资源是否在用户RoleAuthority列表中，未找到则拒绝访问，找到则进入下一步。
    3. 判断客户端请求的方法Method类型，校验所访问的资源的operations内是否存在，未匹配则拒绝访问，匹配则允许访问。
    4. 根据特定数据进行过滤，涉及到的有dns的视图（view）系列以及地址管理的（plan）系列。根据用户所处的角色过滤可见的DNS视图或者地址管理内容。

#### 用户组（UserGroup）
* 顶级资源，包含字段:用户组(Name)、备注(Comment)、用户ID列表(UserIds)、角色ID列表(RoleIds)
* 支持增、删、改、查
* 创建用户组的之前，需要同时获取当前的用户列表以及角色列表。用户组创建时候可选用户以及角色。用户组名称唯一且不能更新。
* 用户组更新获取数据会填充UserIds以及RoleIds分别表示该用户组已绑定的用户以及角色。
* 可更新字段：用户、角色、备注。
* 参数检测：用户以及角色只能选择列表中提供的。

#### 角色（Role）
* 顶级资源，包含字段:角色名(Name)、备注(Comment)、dns视图列表(Views)、IP前缀列表(Plans)
* 支持增、删、改、查
* 角色名称唯一且不能更新。
* 可更新字段：DNS视图列表、IP前缀列表、备注。
* 参数检测：DNS以及IP前缀选择只能选择列表中的。一个角色必须至少拥有一个DNS或者IP前缀权限。

#### 用户（Ddiuser）
* 顶级资源，包含字段:用户名(Name)、密码(Password)、备注(Comment)、角色类型(RoleType)、用户组列表（UserGroupIds）、角色列表（RoleIds）。
* 支持增、删、改、查
* 可更新字段：用户组列表、角色列表、备注。
* 参数检测：用户组以及角色只能选择列表提供中的。用户创建默认密码是123456。
* 一个用户可以不选择角色或者用户组，默认的用户权限为只读，即DNS模块以及地址管理模块看不见任何数据。

#### 白名单（WhiteList）
* 顶级资源，包含字段:IP地址列表（ips）、启用开关（isEnable）
* 支持改、查。
* 参数检测：支持的IP格式:2.2.3.4/24,2.2.2.2,192.168.1.1-192.168.1.6,2001::/32,2001::1-2001::8
