# 用途：  
服主开启一个网站，其它人到这个网站填入自己米游社cookie，服主可以帮忙每天定时签到，签到结果有4种方式可以通知到用户。  
  
# 用法：  
*注意服务器上要有sqlite3，linux一般有，windows上请下载一个放到system32中。*  
先运行`mhySign admin`，在浏览器上访问对应端口网站，创建管理员；管理员登录后做设置，要么让游客能自己注册，要么自己手动添加用户。*注意这个网站开启有时间限制。*  
再运行`mhySign web`，在浏览器上打开对应端口的网站（或用nginx等代理到80端口），普通用户登录后，添加米游社或Hoyolab的Cookie，填写通知的key。  
要帮忙签到时，运行`mhySign sign`，程序将会把数据库中的能签到的用户全签到一次；推荐加入计划任务(windows)或crontab(linux)以每天定时签到。  
  
# 获取mhy的Cookie：  
#### 米游社获取cookie：  
浏览器打开 [米游社·原神](https://www.miyoushe.com/ys/) ，推荐用浏览器的无痕模式（或隐身窗口或InPrivate窗口）打开，点右上角头像登录后，按F12，找到Network(网络)标签页，在上半部分有个Filter输入getUser，此时在中部应该能筛选到 getUserGameUnreadCount 或 getUserFullInfo （如果空白，请刷新一下网页试试），  
点击它，在出现的小框中选择Headers(标头)，向下滚找到Cookie（注意不是Set-Cookie），将它右边的字符串选择复制即可。  
如果觉得这个小框太小了不方便找，请对 getUserGameUnreadCount 右击，Copy，Copy as cURL(bash)，桌面或随便哪里新建一个TXT文本文档，把刚刚复制的粘贴进去，第一行是curl开头，下面的其它行都是 -H 开头，应该只有一行是 -b 开头，这一行后面单引号里的东西就是我们要的Cookie了，复制一下。  
请注意识别，这个cookie里面应该含有 cookie_token 或 cookie_token_v2 字样。  
注意不要点退出登录。  
#### Hoyolab获取cookie：  
浏览器打开 [Hoyolab](https://www.hoyolab.com/home) （记得要FQ，不然不给打开），推荐用浏览器的无痕模式（或隐身窗口或InPrivate窗口）打开，点右上角头像登录后，按F12，找到Network(网络)标签页，在上半部分有个Filter输入getUser，此时在中部应该能筛选到 getUserUnreadCount （如果空白，请刷新一下网页试试），  
接下来操作参考上一条米游社，  
请注意识别，这个cookie里面应该含有 cookie_token 或 cookie_token_v2 字样。  
注意不要点退出登录。  
  
# 获取各种推送通知的key：  
#### 企业微信机器人：  
在企业微信随便找一个你是管理员的群（当然最好偷偷拉2个人成群后再踢掉，这样就是1人群了），右上角点3个点，PC端可以找到“添加群机器人”，手机端点进“群机器人”后再点添加，在机器人信息说明里复制它Webhook地址。  
#### 钉钉机器人：  
（相关操作需要在PC端）在钉钉随便找一个你是管理员的群（当然最好偷偷拉2个人成群后再踢掉，这样就是1人群了），右上角点群设置，在群设置里点进“智能群助手”，点添加机器人，点自定义，添加后，来到它的设置页面，名字随便，消息推送开启，Webhook复制，下面安全设置中如果你服务器有固定IP就填IP，不然就勾自定义关键词，填“签到”，点完成。  
#### 方糖Server酱Turbo版：  
打开 [方糖Server酱](https://sct.ftqq.com/) ，右上角点登录，微信扫码登录，然后点“Key&API”，复制 SendKey 即可。  
#### 方糖Server酱3：  
打开 [方糖Server酱3](https://sc3.ft07.com/) ，右上角点登录，微信扫码登录，然后点“SendKey”，复制 SendKey 即可。  
