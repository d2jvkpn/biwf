## Release
2019-11-17, biwf 1.3.7 和 binx 0.4.14 更新，迁移到 go 1.13 module 组织项目代码和微优化，binx 仍然处于 alpha 阶段。

2019-09-25, biwf 1.3.4, runner.log 中打印 block 结构，允许 block 中任务没有绑定对象（不执行），生成 bash 代码中添加 MAINDIR 变量（替代 dirname $MAIN 命令）。新增 runInContainer.sh 使用容器（docker container）运行 workflow 。

2019-08-31, biwf 1.3.3, 新增 s2j 子命令，可以将 ini 中 section 转化为 json 格式，进一步支持程序间数据交换，同时 project.ini 可以添加除了对象与任务设置之外，普通的 section 用于保存数据（见下方），由于存在 ini 数据读取、格式化重写保存到 log/$RUNNER/project.ini 过程，普通 section  被以伪任务保存在 tcfg 中（section 名字前添加一个空格）。 

[group]

t = t1 t2

n = n1  n2, n3,

=> "group": {"t":["t1","t2"], "n": ["n1  n2", "n3", ""]}


2019-08-23, 更名为 biwf 与 binx，修正生成 project.inin 跨行字符串左对齐错误导致 ini 格式解析错误，运行整个流程（"."）时如果 task 没有匹配的 object 将被移除出 blocks.


2019-08-04, biwf_local 1.3.0 取消 -task、-from，使用 input.Addi 替代，优化 biwf_local list，biwf_web parameter 替换为 var。

2019-07-31, biwf_local 1.2.3 task 配置文件支持添加 .Json 参数，设置值为 true，运行前生成 .json 配置文件（Unit struct），更好地支持与其他程序（jq、rjson 等）进行数据交换，注意所有值都为 string 格式。


2019-07-09, biwf_web v0.4.7，实现 api（query, submit, interrupt, progress, leave），leaf 向 node 发起 RPC 注册（AddLeaf）使用 http 协议，node 生成一个 job 对象与 leaf 构建 tcp 协议的双向 RPC ，demo_local.sh、demo_web.sh 均可在终端 sh 直接运行。API 列表：

GET /api

GET /api/a?item=[ip, leaf, node]

GET /api/b/[$project]/[$version]/query?item=[leaf, record, submit]

GET /api/b/[$project]/[$version]/query?item=[$submitName]

GET /api/b/[$project]/[$version]/progress?item=[$submitName]

GET /api/b/[$project]/[$version]/var?item=[$tk1,$tk2...]

POST /api/b/[$project]/[$version]/interrupt?item=[$submitName]

POST /api/b/[$project]/[$version]/submit?timeout=[$timeout]

POST /api/b/[$project]/[$version]/leave

2019-06-30, biwf_local v.1.1.8, 使用 "exit" 连接 bash 代码与 log（不再保存为单独文件），"print" 模式重命名为 "list"； biwf_web v0.4.2 发布（beta phase），使用双向 RPC (Bi-directional Remote Procedure Call) 构建 leaf-node 连接，leaf 向 node 注册项目（addleaf），node(job) 发送操作指令（query, new, recover, interrupt，progress，leave），leaf 接受指令执行本地任务，node 提供 webpage 与终端用户交互（部分实现），并向 node(job) 发送状态更新。新增第三方依赖 github.com/gin-gonic/gin、github.com/satori/go.uuid。

2019-06-02, v1.1.2 版本，更换项目名为 biwf，使用第一个位置参数替换 -mode，函数优化（支持后续 web 模式），引入状态码：
+ -2: waiting, notstart
+ -1: running
+ 0: done
+ 1: falling, failed
+ 2: cancelled (timeout or interrupted)
+ 3: error


2019-05-15，v1.0.2 版本，DRA（Dynamic Resource Allocation）资源分配优化，并支持进度统计，支持冗余资源分配 —— 运行的 object 任务线数目小于最大并行数，如果单线任务完成，释放的资源重新分配给其他运行中的任务线，加速执行，调整了 DRA、Runner、Block 的关系模型。

2019-05-12, 从 0.6.4 版本升级到 v1.0.0 版本，引入 Dynamic Resource Allocation，实现任务线（block 中的一个 object 依次执行的任务）启动前动态分配资源（CPU、内存），以及在冗余情况下单个任务动态添加资源。


## Introduction
biwf 项目使用 Golang 语言编写，用于构建生物信息 pipeline，ini 格式编写分析任务单元（任务参数 + bash 命令），对象参数、项目参数，包含一套参数设置机制（任务默认参数 -> 全局/项目参数 -> 变动任务参数 -> 对象参数），同时支持并行计算、timeout、任务 log、运行 log，独立保存每个运行任务的脚本与 log，很好地记录任务运行异常 —— 失败、中断或取消。解决 bash 编写流程，难于灵活地传递参数和支持并行计算的问题，Python 编写流程难以阅读与维护问题 —— 逐个命令字符串格式化，然后传递给 bash 执行，以及 n 个参数解析是个糟糕的想法。biwf 依赖的第三方包为 github.com/go-ini/ini。

实现任务的动态资源分配，程序支持从一个已经停止的 runner 恢复运行（recover 模式，忽略已成功运行的任务），test 模式可用于若干个 task 的测试（解除与 pipeline 的绑定）， print 模式用于在终端显示 pipeline 步骤与分组（ini 格式）、打印全局参数（json 格式）、打印任务信息（json 格式），read 模式用于读取 ini 文件中的数据，可用于某些任务中读取项目（在 task 难以导入的）配置参数 —— 如列出全部样本名 $ samples=$($MAIN -mode read -target "log/$RUNNER/project.ini sample.rawdata")。


## biwf_local usage

<pre>
Bioinformatics workflow of "biwf_local", commands:
  run      run tasks of a pipeline
  recover  recover a stopped runner
  test     run single tasks in sequencial
  list     list data of pipeleline in text or json
  read     read a variable in a ini file
  new      create a pipeline program template
  # use "biwf_local  command" for more information about a command
</pre>

## biwf_web usage

<pre>
  $ biwf_web  leaf  <-main mainpath>  [-work ./]  [-pcfg project.ini] \
    [-ip 127.0.0.1]  [-node 9000]  [-port 9001]  [-timeout 0]
    # -main main program path
    # -work project work path

  $ biwf_web  node  [-node 9000]  [-port 90001]  [-data ./biwf_data]
    # -node set node RPC port
    # -port set node HTTP port
    # -data directory to save json files

  -data string
    	set directory to save json (default "./biwf_data")
  -ip string
    	set node ip (default "127.0.0.1")
  -main string
    	set pipeline main program path(biwf_local) in leaf mode
  -node int
    	node RPC service port (default 9000)
  -pcfg string
    	config file to set global varibales, objects and task variables (default "project.ini")
  -port int
    	node HTTP service port (default 9001)
  -timeout string
    	timeout for leaf to keep connected with node (default "0")
  -work string
    	set project workpath for runner (default "./")


</pre>

## ini 格式约束
project.ini、task.cfg、pipeline.ini、global.ini，bash 支持的变量名正则匹配为 "^[_a-zA-Z][_a-zA-Z0-9]*$"，对象名和任务名正则匹配为 "^[_\\.\\-a-zA-Z0-9]*$"。


1. task.cfg 格式
<pre>
[task1]
.Type = sample
var1 = 123
var2 = {{ .k2 }}
.Cmd = ##
    echo "Hello"
    ##
    echo "World"
</pre>
.Type 设置任务绑定的对象类型，值必须是变量、包含 * 前缀的变量、空字符之一。

a. 如果是变量，则在会在脚本中引入对象类型等于匹配对象名（如 sample=t1，t1 对象类型是 sample）；

b. 如果有 * 前缀，怎会引入对象类型等于指定对象（不使用 -object 则标识全部 project.ini 中全部该类型的变量名）的赋值（如 sample="t1 n1 t2 n2"）；

c.如果是空字符，则不引入任何对象，实际上程序自动添加一个空对象，使之与匹配；
var1、var2 必须是变量， {{.k2}} 表示引用全局变量（global.ini 或 project.ini） 的 k2 的值，允许跨行；
.Cmd 第一行和后续的行都不能为空，可以使用 ## 代替空行；


2. global.ini 格式
文件中包含一些列的 key-value 配置全局变量，值允许跨行
<pre>
k1 = v1
    abc
k2 = 123
</pre>

3. pipeline.ini 格式
文件设置了若干个由预设 tasks 组成 pipeline，一个 section 对应一个 pipeline，section 下每个 key 的值为若干任务名（空格或逗号分割），section name 和 key name 与对象名的正则相同
<pre>
[A-1]
step1 = task1 task2
step.2 = task4, task5, task6

[_B3]
step1 = task1, task2, task3
step2 = task6
</pre>
设置 pipeline 为 A1，使用 -task 选择 step1， 则会被自动转换为 task1 task2


4. project.ini 格式
<pre>
## part1
Project = PROJ00001
Version = v1
Pipeline = P1
NC = 40
NG = 32
NP = 4
## part2
k1 = 456
k2 = xxx

## part3
[sample.rawdata]
n1 = rawdata/sample_N1
n2 = rawdata/sample_N2
t1 = rawdata/sample_X1
t2 = rawdata/sample_X2

[somatic.group]
s1 = t1 n1
s2 = t2 n2

## part4
[@task1]
var1 = 789
</pre>

配置文件包含 4 个部分：

part1 中 Pipeline 设置运行的 Project, Version, Pipeline，NC、NG、NP 设置默认总的 CPU 核心数、内存容量（GB）、最大并行数目，（对于 biwf_local, "Project" 与 "Version" 为非必需）；

part2 设置运行全局变量，如果非空字符会覆盖 global 中同名变量的值，也可直接被任务参数引用，part1 中的变量可以全局变量的形式引用。注意如果值为空字符，且在 globlal.ini 中已经设置，则为无效设置，不会覆盖或更新；

part3 设置对象参数，section name 为对象类型（变量名）与对象属性（变量名），使用点号分隔，以 [sample.rawdata] 为例，如果某个任务 .Type 设置为 sample，且包含 rawdata 变量，且档任务与对象组合时，对象的 rawdata 值会覆盖任务的默认 rawdata 值，section name 添加类似 @tk 后缀，则改设置只作用于特定 task，如 [sample.evalue@blast]；

part4 的 section name 为任务名添加 @ 前缀，包含的 key 为任务的预设变量，如果为非空字符，则会更新任务的变量值。


