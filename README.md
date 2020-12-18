# Tieba-Sign-Actions
基于Github Actions实现的无服务器永久免费云签
- [x] 多线程
- [x] 贴吧签到   
- [x] 知道签到
- [x] 文库签到
- [x] 名人堂助攻
- [x] 云灌水（测试中）
- [ ] 云封禁（计划中）
- [x] 一日三次签到（0点，12点，22点）
- [x] 特殊吧补签，防止漏签
- [x] 签到结果通知：1. 电报（telegram）2. server酱（微信）

Demo：https://tb-act.tk
# 使用说明
## 注册一个github账号，已有请跳过
[注册教程](https://jingyan.baidu.com/article/86fae346e723303c49121abb.html)
## 一、Fork此仓库
打开https://github.com/libsgh/Tieba-Sign-Actions
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/1.png)
## 二、设置 BUDSS
### BDUSS 获取
1. 电脑浏览器（例如chrome）打开百度首页并登录(最好用隐身模式，防止退出登录导致bduss失效)，或打开开发者模式network中查找百度首页请求，在右侧的请求中找到cookie中的bduss并复制
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-1-1.gif)
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-1-2.png)
2. 在线获取工具,扫码方式更安全
- http://bduss.imyfan.com/
- https://noki.top/bduss
- http://tool.cccyun.cc/tool/bduss/index2.html
### 设置 Secrets
Secrets对照表

秘钥名称 | 说明 |  是否可选 
-|-|-
BDUSS | 签到cookie参数 | 必填 |
GH_TOKEN | github的access_token，用于上传签到结果 | 可选 |
~~OWNER_REPO~~ | ~~云签仓库名字，格式：xxx/Tieba-Sign-Actions~~ | 改为自动获取，不需要配置 |
TELEGRAM_APITOKEN | 电报机器人api_token | 可选 |
TELEGRAM_CHAT_ID | 电报通知的user_id | 可选 |
AUTH_AES_KEY | 查询详情的AES秘钥，16位，由字母和数字组成，填写此选项程序将开启签到详情记录, [随机密码生成器](https://suijimimashengcheng.51240.com/)| 可选 |
HOME_URL | 你的云签网址，注意地址末尾请不要输入"/"，用于发送查看签到详情的地址 | 可选 |
SCKEY | 采用[Server酱](http://sc.ftqq.com/)推送签到结果的秘钥 | 可选 |
NOTIFY_COUNT  | 通知/签到数据更新 次数，默认每天通知一次（第一次），不影响多次签到，只是不再更新或通知结果 | 可选 |

1. 打开Tieba-Sign-Actions > Settings > Secrets > New secret
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-2-1.png)
2. 输入上一步获取的BDUSS，**每行一个BDUSS，对应一个签到账号**
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-2-2.png)
3. 【结果通知，可选】添加**TELEGRAM_APITOKEN**,telegram的机器人的api token

    如何创建telegram机器人，请参考：https://blog.csdn.net/weixin_42776979/article/details/88239086
4. 【结果通知，可选】添加**TELEGRAM_CHAT_ID**,telegram的通知的用户ID，可以在@getidsbot中输入<code>/start</code>获取
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-2-4.png)
5. 【云签网页，可选】利用github pages为云签添加首页，用来随时查看签到情况（支持手机、pc访问）
- 为项目开启github pages，根据自己情况可以使用自定义域名或是github提供的二级域名
  ![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-2-5-1.png)
  例如：http://tb-act.tk、https://libsgh.github.io/Tieba-Sign-Actions/
  
- 添加Secret: **GH_TOKEN**，github的access_token，用于上传签到结果到github仓库
- ~~添加Secret: **OWNER_REPO**，云签仓库的名称，例如我的是**libsgh/Tieba-Sign-Actions**~~(改为自动获取)
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/2-2-5-3.jpg)
 
## 三、启用 Action
1. 点击**Action**，再点击**I understand my workflows, go ahead and enable them**  
2. 修改任意文件后提交一次
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/3.png)
## 四、查看运行结果
Actions > Tieba-Sign-Actions
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/4-1.png)
![avatar](https://cdn.jsdelivr.net/gh/libsgh/Tieba-Sign-Actions@master/doc/4-2.png)

此后，将会在每天00:00、12:00、22:00各执行一次签到（注意服务器时区+8小时是执行的北京时间）

~~修改任一文件并push就会触发一次签到~~
点击**star**按钮手动触发执行签到

若有需求，可以在[.github/workflows/run.yml]中自行修改

如果要停止**Action**，请删除**Fork**,如果要终止签到任务请删除**Secrets**中的**BDUSS**