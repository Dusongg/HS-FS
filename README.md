> - language: go1.22.5 windows/**amd64**
>
> - IDE：Goland
>
> - lib：
>   - https://github.com/lxn/walk
>   - https://github.com/akavel/rsrc
>
> - OS：Windows 

# CookBook

共两种搜索模式：

1. 原文本搜索：搜索的文件为xml格式的文本文件，且搜索内容包含注释
   - 搜索速度：单次搜索与第二种相比较慢
2. 处理后的文本搜索：处理后的文本仅包含xml文件中code标签部分，且可选择是否带注释
   - 搜索速度：最开始需要1min时间处理，之后单次搜索与第一种相比较快



一次搜索的大概流程（以搜索关键字`hs_strcpy`为例）

![image-20240725092536621](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240725092536621.png)



# 1 原文本搜索

## 1.1搜索步骤

> 原文本搜索意味着：搜索的文件为xml格式的文本文件，且搜索内容包含注释

![image-20240724164235239](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724164235239.png)

此时看到弹窗

![image-20240724164335170](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724164335170.png)

点击确定，出现下面窗口，可点击Browser浏览并输入本地的==业务逻辑==和==原子==文件所在目录

![image-20240724164356554](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724164356554.png)

例如：

![image-20240724164621392](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724164621392.png)

输入完成后点击OK按钮，等待0-3秒结果会展示出来

![image-20240724164730813](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724164730813.png)

## 1.2 原文本搜索下的重新解析按钮

业务逻辑和原子两大文件中有文件的增删改动时（文件内容改动不影响），需要点击重新加载按钮，然后再点击Run运行，在这之后如果没有增删改动则不需要重复点击

![image-20240724165248196](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724165248196.png)

# 2 处理后的文本搜索

> 处理后的文本仅包含xml文件中code标签部分，且可选择是否带注释

## 2.1 搜索步骤

![image-20240724165617207](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724165617207.png)

点击生成文件之后会显示进度条

![image-20240724165701844](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724165701844.png)

完成后，点击Run，得到如下结果

![image-20240724165804498](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724165804498.png)

## 2.2 处理后文本搜索下的重新解析按钮

业务逻辑和原子两大文件中有文件的增删改动时（文件内容改动不影响），需要点击重新加载按钮，然后再点击Run运行，在这之后如果没有增删改动则不需要重复点击

![image-20240724170035091](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170035091.png)

点击后出现设置界面，再次点击生成文件即可

![image-20240724170016833](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170016833.png)



# 3 其他功能

## 3.1 双击搜索结果打开文件

双击搜索结果会打开调用链的最后一个节点所对应的文件

对于非原文本搜索可选择打开解析前或者解析后的文件

![image-20240724170248787](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170248787.png)

## 3.2 鼠标右键点击结果复制文本



## 3.3 去重

![image-20240724170418325](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170418325.png)

对于非去重的结果可能会出现以下情况，即LS调用的LF会在结果中再展示一行，去重之后图中第二个方框中的结果则不会显示

![ae68fe70c6121e21cf348e0aed86b2f3](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/ae68fe70c6121e21cf348e0aed86b2f3.png)



## 3.4 导出结果

将所有调用链结果保存到文本中

![image-20240724170658734](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170658734.png)



## 3.5 历史搜索记录

![image-20240724170814596](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724170814596.png)



## 3.6 正则匹配

结果的调用链的最后一个节点会展示匹配到的字符串，每个字符串对应目标所在行的行号

![image-20240724172340549](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724172340549.png)

## 3.7 结果排序

![image-20240724171204370](https://typora-dusong.oss-cn-chengdu.aliyuncs.com/image-20240724171204370.png)

# 4 报错解释

搜索结果报错`not found`说明该调用链的最后一个节点文件在业务逻辑和原子这两个目录下没有找到，（很大原因是勾选了原文本搜索，导致搜索到了很多没有去注释的代码）