# CookBook

## 1 预处理文件

### 1.1 Parse

1. 点击Parse，打开预处理菜单，(预处理完的文件放在output目录下，该目录位于在HS_FS文件目录下)

![image-20240717155105554](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717155105554.png)

- Reload：清空output文件夹，重新加载
- Append：向output文件夹里新增文件

2. 输入待处理的文件或目录

### 1.2 自定义输出文件路径

1. 默认输出路径

![image-20240717155019007](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717155019007.png)

2. 自定义输出路径

![image-20240717155247789](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717155247789.png)

该路径会保存到指定文件（outputdir.txt），用于后续寻找

![image-20240717155336156](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717155336156.png)

### 1.3（可选）Browser使用说明

![image-20240717101417481](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717101417481.png)

> [!NOTE]
>
> 预处理任务耗时较长，请耐心等待



## 2 搜索

1. 输入路径与搜索目标（拖拽/输入/点击Browser浏览）

![image-20240717100848585](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717100848585.png)

2. 选择匹配模式（正则匹配功能尚未实现）
3. 点击Run运行

![image-20240717102137680](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717102137680.png)

> [!CAUTION]
>
> 目标所在行展示的是调用链的最后一个结点，且为预处理后的文件(位于output目录)中的目标所在行数



## 3 其他功能与说明

### 3.1单击搜索结果打开调用链的最后一个结点所对应的文件

![image-20240717102523850](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240717102523850.png)

### 3.2 其他说明

- 搜索采用单线程的dfs，效率一般，若遇到搜索耗时较多的情况可分成多个子目录依次搜索
  - 如”DevCodes\经纪业务运营平台V21\业务逻辑\债券“该目录耗时较长，可改为"DevCodes\经纪业务运营平台V21\业务逻辑\债券\债券交易"等其子目录依次搜索
- 搜索过程中处理了==函数循环调用==与==调用函数文件不存在==问题，将其以报错形式展示在界面的报错框中
- 对于一个文件的其中一行搜索，只考虑了一行内仅有一个[AF]/[AS]/[AP]/[LS]/[LF]的情况



## 4 程序后续优化与设计

1. 采用协程优化预处理与搜索速度
2. 完成正则匹配功能



> Dusong   7/17/2024









> Dusong 7/22/2024

# 测试

![image-20240722092128570](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240722092128570.png)