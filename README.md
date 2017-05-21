# 多版本备份工具



源文件夹：指待备份文件夹。

备份文件夹：指备份数据所在的文件夹，所有命令应该在此文件夹下执行。

版本号：

1. 数字版本号。如v1代表第一个版本，v2代表第二个版本，v-1代表最后一个版本，v-2代表倒数第二个版本，版本根据时间先后顺序排序。
2. SHA1版本号，支持短格式。如da39a3ee5e6b4b0d3255bfef95601890afd80709，如果在所有版本中以da39开头的只有这一个版本，那么da39即可作为此版本的短版本号。
3. 时间戳版本号，支持短格式。如20060102150405，但如果同一时间有2个或以上版本，则不能使用时间戳版本号，短格式的定义同上。





## 1.基本原理

备份原理及存储参考了Git。

每次备份时会提取源文件夹下所有的文件及文件夹的元数据（文件路径、最后修改时间、文件大小、文件SHA1），生成快照。并保存新增和修改的文件。

所有数据存储在备份文件夹下的index文件和objects目录中。index是版本索引文件，包括快照SHA1、时间戳。objects存储所有的文件及快照。



## 2.使用

### 2.1 初始化

```shell
mvb init /Users/whow/git/mvb/src
```

* ```mvb init [源文件夹]``` 初始化备份文件夹 。如果源文件夹路径移动了，重新执行此命令。





### 2.2 备份

```shell
mvb backup
```

* ```mvb backup``` 备份源文件夹。如果没有任何变化，不会执行任何操作。执行成功后将输出新版本SHA1版本号。




### 2.3 还原

```shell
mvb restore
mvb restore da39
mvb restore 20060102150405
mvb restore v-1 /temp
```

* ```mvb restore``` 还原最新版本到源文件夹，与 ```mvb restore v-1``` 相同。
* ```mvb restore [版本号]``` 还原指定版本到源文件夹。
* ```mvb restore [版本号] [目标文件夹]``` 还原指定版本到目标文件夹。




### 2.4 链接

```shell
mvb link v-1 /temp
```

```mvb link [版本号] [目标文件夹]``` 与**还原**命令第三种格式相似，不过使用符号链接方式替代了文件拷贝。目标文件夹必须存在且为空。



### 2.5 版本列表

```shell
mvb list
mvb list v1
mvb list da39
mvb list 2006
```

* ```mvb list``` **倒序**输出所有版本信息（SHA1、时间戳）。
* ```mvb list [版本号]``` 输出所有匹配的版本信息。




### 2.6 获取内容

```shell
mvb get
mvb get v-1
mvb get v-1 mvb/
mvb get v-1 mvb/app.go
```

* ```mvb get``` **倒序**获取所有版本信息（SHA1、时间戳），同 ```mvb list``` 。
* ```mvb get [版本号]``` 获取指定版本快照信息（文件列表，包括文件夹及文件，文件信息包括文件路径、最后修改时间、文件大小、文件SHA1）。
* ```mvb get [版本号] [文件夹]``` 获取指定版本文件夹下所有下级文件夹及文件列表信息，文件夹名最后需带上/。
* ```mvb get [版本号] [文件]``` 获取指定版本文件内容。





### 2.7 删除

```shell
mvb delete v-1
mvb delete da39
mvb delete 2006
```

* ```mvb delete [版本号]``` 删除所有匹配版本，匹配版本可通过 ```mvb list [版本号]``` 查询。


为防止误操作，没有提供 ```mvb delete``` 命令删除所有版本，不过可以通过清空或删除index文件实现，或者替代方案为 ```mvb delete 2``` ，2作为时间戳短版本号事实上匹配所有版本。



### 2.8 比较

```shell
mvb diff
mvb diff v1
mvb diff v1 v2
```

* ```mvb diff``` 比较最新备份版本与源文件夹差异。
* ```mvb diff [版本号]``` 比较指定版本与源文件夹差异。
* ```mvb diff [版本号1] [版本号2]``` 比较2个版本之间的差异。

比较结果：

```shell
* mvb.go
+ mvb/app.go
- mvb/object.go
```

* ```*``` 表示文件内容有变化。
* ```+``` 表示文件新增。
* ```-``` 表示文件删除。



### 2.9 预览

```shell
mvb preview
```

* ```mvb preview``` 预览源文件夹快照信息，包括文件列表及快照SHA1版本号。



### 2.10 校验

```shell
mvb check
```

* ```mvb check``` 校验备份数据完整性。



### 2.11 文件回收

```shell
mvb gc
```

* ```mvb gc``` 将删除所有没有用到的文件。

执行删除命令时，只是从索引中将版本信息删除，版本快照及文件数据还存储在objects中，将会产生垃圾文件。文件回收命令将遍历索引文件及版本快照，找出所有有用的文件，删除所有无用文件。



## 3.实现

```shell
# cat index
e43f8056e27ca8002d8df2dbc796936fc7fc7293 20170520235321+0800
0a00790d7df7d2bfddd1761aaf024f0f5f5b8906 20170521000308+0800
8a4f82f0dc2c5fecaa9718e8c897bbdc04299239 20170521003825+0800

# find objects -type f | head
objects/00/d01472e88aea8177c87b0043a268683410dafb
objects/04/6fbc692c58d2a8dd4d58b74c83d25da334d10b
objects/05/32850fc9094c340cefbb965cb2b72bcfa632d0
objects/07/eb058df792610fa037509052fa6eee92168158
objects/0a/00790d7df7d2bfddd1761aaf024f0f5f5b8906
objects/0a/ab618964b9c36ad833f4174247975746b00494
objects/0a/bc71b05a8437042f4f512e53554945d37125f7
objects/0f/19178159773986c765aa63b91d66e312865334
objects/0f/57f2f7f56896eda8dfe42ca6e266001610c8ca
objects/12/bed672ffaba527eeebd757c1843797622a2afe

# cat objects/8a/4f82f0dc2c5fecaa9718e8c897bbdc04299239 | tail -6
51b599148ca2b64341d1d744bf106758228aa8ce 20170521003521+0800                8983 mvb.go
                                         20170521003521+0800                     mvb/
0f19178159773986c765aa63b91d66e312865334 20170520213725+0800                 263 mvb/app.go
68e64f592acc017e510296bb444c0ee30ba3d5c2 20170521001544+0800                3770 mvb/core.go
175a62203accd240896a92c1198853d2f4683730 20170521003521+0800                5038 mvb/index.go
b938cab7436dd0c5a38b39583527c51b34cc0049 20170521003302+0800                3946 mvb/objects.go

# cat objects/0f/19178159773986c765aa63b91d66e312865334
package mvb

import (
        "fmt"
        "os"
)

var Verbose bool

func Errorf(format string, a ...interface{}) {
        fmt.Fprintf(os.Stderr, format, a...)
        os.Exit(1)
}

func Verbosef(format string, a ...interface{}) {
        if Verbose {
                fmt.Fprintf(os.Stdout, format, a...)
        }
}
```

数据存储在index文件和objects文件夹中。objects中存放文件数据及版本快照。

index文件是文本格式，每行都是一个版本，按照时间正序排序。行数据格式为40位版本快照SHA1、空格分隔、19位时间戳。

版本快照也是文本格式，存储在objects中，每行都是一个文件或文件夹的元数据，按相对路径正序排序。数据格式为40位版本快照SHA1、空格分隔、19位时间戳、空格分隔、19位文件大小、空格分隔、相对路径。文件夹的SHA1及文件大小为空，文件夹相对路径后添加/。

objects文件夹内文件路径由文件SHA1生成，第一层目录为SHA1头2位，目录内文件名为SHA1后38位。

计算源文件夹内所有文件SHA1时，使用最新版本快照加快计算速度，当文件的路径、最后修改时间、文件大小相同时，直接使用快照中的SHA1值。对于需要读取内容计算的文件，使用Goroutines并发执行。

由于版本快照中文件已按路径排序，可以使用二分搜索。

在拷贝文件时，需将最后修改时间同时拷贝。

版本快照直接全部加载到内存中，所以当文件较多时内存占用较大，不过暂时满足需求，后续可再做优化。

当前版本尚未经过严格测试。