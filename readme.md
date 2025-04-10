# 用法：  
先运行`mhySign admin`，在浏览器上打开，创建管理员；管理员登录后做设置，要么让游客能自己注册，要么自己手动添加用户。  
再运行`mhySign web`，在浏览器上打开，普通用户登录后，添加米游社或Hoyolab的Cookie，填写通知key。  
要签到时，运行`mhySign sign`，程序将会把数据库中的能签到的用户全签到一次；推荐加入计划任务(windows)或crontab(linux)以每天定时签到。  
  
### 米游社获取cookie：  
浏览器打开 [米游社·原神](https://www.miyoushe.com/ys/) ，点右上角头像登录后，按F12，在Console（控制台）中输入`document.cookie`，回车，在出现的字符串上右击，Copy string contents，就复制好了。  
注意不要点退出登录。  
### Hoyolab获取cookie：  
浏览器打开 [Hoyolab](https://www.hoyolab.com/home) （记得要FQ），点右上角头像登录后，按F12，在Console（控制台）中输入`document.cookie`，回车，在出现的字符串上右击，Copy string contents，就复制好了。  
注意不要点退出登录。  
  
### 企业微信机器人：  
在企业微信随便找一个你是管理员的群（当然最好偷偷拉2个人成群后再踢掉，这样就是1人群了），右上角点3个点，PC端可以找到“添加群机器人”，手机端点进“群机器人”后再点添加，在机器人信息说明里复制它Webhook地址。  
### 钉钉机器人：  
（相关操作需要在PC端）在钉钉随便找一个你是管理员的群，右上角点群设置，在群设置里点进“智能群助手”，点添加机器人，点自定义，添加后，来到它的设置页面，名字随便，消息推送开启，Webhook复制，下面安全设置中如果你服务器有固定IP就填IP，不然就勾自定义关键词，填“签到”，点完成。  
### 方糖Server酱：  
（Server酱3未适配，只有老Server酱）打开 [方糖Server酱](https://sct.ftqq.com/) ，右上角点登录，微信扫码登录，然后点“Key&API”，复制 SendKey 即可。  
 
