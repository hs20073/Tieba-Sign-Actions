var $$ = mdui.JQ;
var key = getQueryVariable("k");
var href = window.location.href;
var uid = getQueryVariable("uid");
var pageNo = 1;
var data = null;
var status = 0;
var cdnDataUrl = "";
if(!key){
    if($.cookie("tsa_pwd") == null){
        var value = prompt("访问受到限制，您需要提供访问密码才能查看。","");
        if (value != null && value != ""){
            $.cookie("tsa_pwd", value);
            window.location.href = changeURLArg(href,'k', value);
        }
    }else{
        key=$.cookie("tsa_pwd");
        uid = initMenu(key, uid);
        initDetail(pageNo);
    }

}else{
    uid = initMenu(key, uid);
    initDetail(pageNo);
}
function getDateDiff (dateTimeStamp) {
    var result = ''
    var minute = 1000 * 60
    var hour = minute * 60
    var day = hour * 24
    var month = day * 30
    var now = new Date().getTime()
    var diffValue = now - dateTimeStamp
    if (diffValue < 0) return
    var monthC = diffValue / month
    var weekC = diffValue / (7 * day)
    var dayC = diffValue / day
    var hourC = diffValue / hour
    var minC = diffValue / minute
    if (monthC >= 1) {
        result = "" + parseInt(monthC) + "月前"
    }
    else if (weekC>=1) {
        result = "" + parseInt(weekC) + "周前"
    }
    else if (dayC >= 1) {
        result = ""+ parseInt(dayC) + "天前"
    }
    else if (hourC >= 1){
        result = "" + parseInt(hourC) + "小时前"
    }
    else if (minC >= 1) {
        result = ""+ parseInt(minC) + "分钟前"
    } else {
        result = "刚刚"
    }
    return result
}

function getDuration(t) {
    if(t < 60000){
        return parseInt((t % 60000 )/1000)+"秒";
    }else if((t>=60000)&&(t<3600000)){
        return getString(parseInt((t % 3600000)/60000))+":"+getString(parseInt((t % 60000 )/1000));
    }else {
        return getString(parseInt(t / 3600000))+":"+getString(parseInt((t % 3600000)/60000))+":"+getString(parseInt((t % 60000 )/1000));
    }
}
function getString(t) {
    var m = "";
    if(t > 0){
        if(t<10){
            m="0"+t;
        }else{
            m=t+"";
        }
    }else{
        m="00";
    }
    return m;
}
$$("#inputkey").on("keydown", function (event) {
    if (event.keyCode === 13) {
        var key = $$("#inputkey").val();
        initDetail(pageNo);
    }
});
function initMenu(key, uid){
    $$.ajax({
        method: 'GET',
        url: 'data/users.txt',
        async:false,
        success: function (data) {
            var dataText = aseDecrypt(data, key);
            data = JSON.parse(dataText);
            $$("#tb-list").empty();
            if(!uid){
                uid = data[0].uid;

            }
            $$.each(data, function (i, item) {
                var url = "/tb.html?k="+key+"&uid="+item.uid
                var html = "";
                var active = "";
                if(uid == item.uid){
                    active = "mdui-list-item-active";
                    cdnDataUrl = item.cdn_data_url;
                }
                html += "<li class=\"mdui-list-item mdui-ripple "+active+"\" onclick=\"javascript:location.href=\'"+url+"\'\">"+
                "                <div class=\"mdui-list-item-avatar\"><img src=\""+item.head_url+"\"/></div>"+
                "                <div class=\"mdui-list-item-content\">"+
                "                    <div class=\"mdui-list-item-title\">"+item.name+"</div>"+
                "                </div>"+
                "</li>";
                $$("#tb-list").append(html);
            });
        }
    });
    return uid;
}
function initDetail(pageNo){
    if(data == null){
        $$.ajax({
            method: 'GET',
            cache:false,
            async:false,
            url: cdnDataUrl,
            success: function (result) {
                data = result;
                commBuild(pageNo);
            }
        });
    }else{
        commBuild(pageNo);
    }
}
function excepQuery(s) {
    status = s;
    commBuild(1)
}
function commBuild(pageNo){
    var searchKey = $$("#inputkey").val();
    try {
        var dataText = aseDecrypt(data, key);
        var d = JSON.parse(dataText);
        var loadData =  d.data[pageNo-1].list
        if(searchKey != "" || status == 1 || status == 2){
            loadData = [];
            $$.each(d.data, function(i, item){
                $$.each(item.list, function(i, record){
                    if(searchKey != "" && record.name.indexOf(searchKey) == -1) {
                        return true;
                    }
                    if(status == 1 && (record.error_code == "0" || record.error_code == "160002" || record.error_code == "2280007" || record.error_code == "340011" || record.error_code == "1989004")) {
                        return true;
                    }
                    if(status == 2 && (record.error_code != "2280007" && record.error_code != "340011" && record.error_code != "1989004")) {
                        return true;
                    }
                    loadData.push(record);
                });
            });
        }
        if(searchKey == "" && status == 0){
            var lastDisable = "";
            if(pageNo == 1){
                lastDisable = "disabled";
            }
            var nextDisable = "";
            if(pageNo == d.data[0].total_pages){
                nextDisable = "disabled";
            }
            $$("#userInfo").html(d.user.name+" <small>关注 "+d.user.total+"</small>");
            $$("#page_footer").empty();
            $$("#page_footer").append("<button class=\"pageBtn mdui-btn mdui-btn-raised mdui-ripple\" onclick=\"initDetail(1)\">首页</button>&nbsp;&nbsp;");
            $$("#page_footer").append("<button class=\"pageBtn mdui-btn mdui-btn-raised mdui-ripple\" "+lastDisable+" onclick=\"initDetail("+(pageNo-1)+")\">上一页</button>&nbsp;&nbsp;");
            if(!(navigator.userAgent.match(/(phone|pad|pod|iPhone|iPod|ios|iPad|Android|Mobile|BlackBerry|IEMobile|MQQBrowser|JUC|Fennec|wOSBrowser|BrowserNG|WebOS|Symbian|Windows Phone)/i))) {
                $$.each(d.data[(pageNo-1)].pages, function (i, item) {
                    var active = "";
                    if(item == pageNo){
                        active = "mdui-btn-active";
                    }
                    $$("#page_footer").append(" <button class=\"pageBtn mdui-btn mdui-btn-raised mdui-ripple "+active+"\" onclick=\"initDetail("+item+")\">"+item+"</button>");
                });
            }
            $$("#page_footer").append("<button class=\"pageBtn mdui-btn mdui-btn-raised mdui-ripple\" "+nextDisable+" onclick=\"initDetail("+(pageNo+1)+")\">下一页</button>&nbsp;&nbsp;");
            $$("#page_footer").append("<button class=\"pageBtn mdui-btn mdui-btn-raised mdui-ripple\" onclick=\"initDetail("+d.data[(pageNo-1)].total_pages+")\">尾页</button>&nbsp;&nbsp;");
        }else{
            $$("#page_footer").empty();
        }
        if((navigator.userAgent.match(/(phone|pad|pod|iPhone|iPod|ios|iPad|Android|Mobile|BlackBerry|IEMobile|MQQBrowser|JUC|Fennec|wOSBrowser|BrowserNG|WebOS|Symbian|Windows Phone)/i))) {
            $$('#mobel_table').show();
            $$('#pc_table').hide();
            $$('#mobel_table').empty();
            var html="";
            $$.each(loadData, function (i, item) {
                html += "			  <div class=\"mdui-panel-item\">";
                html += "			    <div class=\"mdui-panel-item-header\">";
                html += "			      <div class=\"mdui-list-item-avatar\"><img src=\""+item.avatar+"\"\/><\/div>";
                html += "			      <div class=\"mdui-list-item-content\">"+item.name+"<\/div>";
                html += "			      <a href=\"javascript:void(0);\" class=\"";
                if(item.level_id=="1"||item.level_id=="2"||item.level_id=="3"||item.level_id=="4"){
                    html+="d_badge_icon1";
                }else if(item.level_id=="5"||item.level_id=="6"||item.level_id=="7"||item.level_id=="8"||item.level_id=="9"){
                    html+="d_badge_icon2";
                }else if(item.level_id=="10"||item.level_id=="11"||item.level_id=="12"||item.level_id=="13"||item.level_id=="14"||item.level_id=="15"){
                    html+="d_badge_icon3";
                }else if(item.level_id=="16"||item.level_id=="17"||item.level_id=="18"){
                    html+="d_badge_icon4";
                }
                html += "						\">";
                html += "							<div class=\"d_badge_lv\">"+item.level_id+"<\/div>";
                html += "					<\/a>";
                html += "			    <\/div>";
                html += "			    <div class=\"mdui-panel-item-body\">";
                html += "			     <p class=\"mdui-text-color-deep-purple-900\"><b>经验<\/b>: "+item.cur_score+"/"+item.levelup_score+"<\/p>";
                if(item.error_code == "0"){
                    html += "					      <p class=\"mdui-text-color-green\" style=\"font-weight: 900\"><b>签到状态<\/b>: 签到成功<\/p>";
                }else if(item.error_code == "160002"){
                    html += "					      <p class=\"mdui-text-color-blue-900\" style=\"font-weight: 900\"><b>签到状态<\/b>: "+item.ret_msg+"<\/p>";
                }else if(item.error_code == "2280007" || item.error_code == "340011" || item.error_code == "1989004"){
                    html += "					      <p class=\"mdui-text-color-teal-900\" style=\"font-weight: 900\"><b>签到状态<\/b>: 已补签<\/p>";
                }else{
                    html += "					      <p class=\"mdui-text-color-red\" style=\"font-weight: 900\"><b>签到状态<\/b>: "+item.ret_msg+"<\/p>";
                }
                html += "			      <p><b>签到时间<\/b>："+getDateDiff(item.sign_time)+"";
                html += "				  <\/p>";
                html += "			    <\/div>";
                html += "			  <\/div>";
            });
            $$('#mobel_table').html(html);
            mdui.mutation();
        }else{
            $$('#mobel_table').hide();
            $$('#pc_table').show();
            var htmStr = ""
            var index = 0;
            $$.each(loadData, function (i, item) {
                var html =" <tr>"+
                    "            <td>"+((pageNo-1)*15+index+1)+
                    "            </td>"+
                    "            <td>"+
                    "                <div class=\"mdui-chip\">"+
                    "                    <img class=\"mdui-chip-icon\" src=\""+item.avatar+"\"/>"+
                    "                    <span class=\"mdui-chip-title\">"+
                    item.name+
                    "                    </span>"+
                    "                </div>"+
                    "            </td>"+
                    "            <td><a href=\"javascript:void(0);\" class=\"";
                if(item.level_id=="1"||item.level_id=="2"||item.level_id=="3"||item.level_id=="4"){
                    html+="d_badge_icon1";
                }else if(item.level_id=="5"||item.level_id=="6"||item.level_id=="7"||item.level_id=="8"||item.level_id=="9"){
                    html+="d_badge_icon2";
                }else if(item.level_id=="10"||item.level_id=="11"||item.level_id=="12"||item.level_id=="13"||item.level_id=="14"||item.level_id=="15"){
                    html+="d_badge_icon3";
                }else if(item.level_id=="16"||item.level_id=="17"||item.level_id=="18"){
                    html+="d_badge_icon4";
                }
                html +="\"><div class=\"d_badge_lv\">"+item.level_id+"</div></a>"+
                    "</td>"+
                    "            <td>"+item.cur_score+"/"+item.levelup_score+"</td>"+
                    "            <td>";
                if(item.error_code == "0"){
                    html+="<div class=\"mdui-text-color-green\" style=\"font-weight: 900\">签到成功</div>"
                }else if(item.error_code == "160002"){
                    html+="<div class=\"mdui-text-color-blue\" style=\"font-weight: 900\">"+item.ret_msg+"</div>"
                }else if(item.error_code == "2280007" || item.error_code == "340011" || item.error_code == "1989004"){
                    html+="<div class=\"mdui-text-color-teal\" style=\"font-weight: 900\">已补签</div>"
                }else{
                    html+="<div class=\"mdui-text-color-red\" style=\"font-weight: 900\">"+item.ret_msg+"</div>"
                }
                html+= "</td>"+
                    "            <td>"+getDateDiff(item.sign_time)+"</td>"+
                    "        </tr>";
                htmStr+=html;
                index++;
            });
            $$('#pc_body').html(htmStr);
        }
    }catch(err){
        //在此处理错误
    }
}
function aseDecrypt(msg, key) {
    key = PaddingLeft(key, 16);//保证key的长度为16byte,进行'0'补位
    key = CryptoJS.enc.Utf8.parse(key);
    var encryptedHexStr = CryptoJS.enc.Hex.parse(msg);
    var srcs = CryptoJS.enc.Base64.stringify(encryptedHexStr);
    // key 和 iv 使用同一个值
    var decrypted = CryptoJS.AES.decrypt(srcs, key, {
        iv: key,
        mode: CryptoJS.mode.CBC,// CBC算法
        padding: CryptoJS.pad.Pkcs7 //使用pkcs7 进行padding 后端需要注意
    });
    var decryptedStr = decrypted.toString(CryptoJS.enc.Utf8);
    var value = decryptedStr.toString();
    return value;
}

function aseEncrypt(msg, key) {
    key = PaddingLeft(key, 16);//保证key的长度为16byte,进行'0'补位
    key = CryptoJS.enc.Utf8.parse(key);
    // 加密结果返回的是CipherParams object类型
    // key 和 iv 使用同一个值
    var encrypted = CryptoJS.AES.encrypt(msg, key, {
        iv: key,
        mode: CryptoJS.mode.CBC,// CBC算法
        padding: CryptoJS.pad.Pkcs7 //使用pkcs7 进行padding 后端需要注意
    });
    // ciphertext是密文，toString()内传编码格式，比如Base64，这里用了16进制
    // 如果密文要放在 url的参数中 建议进行 base64-url-encoding 和 hex encoding, 不建议使用base64 encoding
    return  encrypted.ciphertext.toString(CryptoJS.enc.Hex)  //后端必须进行相反操作

}
// 确保key的长度,使用 0 字符来补位
// length 建议 16 24 32
function PaddingLeft(key, length){
    let  pkey= key.toString();
    let l = pkey.length;
    if (l < length) {
        pkey = new Array(length - l + 1).join('0') + pkey;
    }else if (l > length){
        pkey = pkey.slice(length);
    }
    return pkey;
}
function getQueryVariable(variable) {
    var query = window.location.search.substring(1);
    var vars = query.split("&");
    for (var i=0;i<vars.length;i++) {
        var pair = vars[i].split("=");
        if(pair[0] == variable){return pair[1];}
    }
    return(false);
}
function changeURLArg(url,arg,arg_val){
    var pattern=arg+'=([^&]*)';
    var replaceText=arg+'='+arg_val;
    if(url.match(pattern)){
        var tmp='/('+ arg+'=)([^&]*)/gi';
        tmp=url.replace(eval(tmp),replaceText);
        return tmp;
    }else{
        if(url.match('[\?]')){
            return url+'&'+replaceText;
        }else{
            return url+'?'+replaceText;
        }
    }
}
function test(){
    $$('#mobel_table').show();
    $$('#pc_table').hide();
    var ss = '<div class="mdui-panel-item">\t\t\t    <div class="mdui-panel-item-header">\t\t\t      <div class="mdui-list-item-avatar"><img src="http://imgsrc.baidu.com/forum/pic/item/77c6a7efce1b9d168ab0bb32f0deb48f8c546469.jpg"/></div>\t\t\t      <div class="mdui-list-item-content">艾弗森</div>\t\t\t      <a href="javascript:void(0);" class="d_badge_icon3\t\t\t\t\t\t">\t\t\t\t\t\t\t<div class="d_badge_lv">13</div>\t\t\t\t\t</a>\t\t\t    </div>\t\t\t    <div class="mdui-panel-item-body">\t\t\t     <p class="mdui-text-color-deep-purple-900"><b>经验</b>: 16358/18000</p>\t\t\t\t\t      <p class="mdui-text-color-blue-900" style="font-weight: 900"><b>签到状态</b>: 亲，你之前已经签过了</p>\t\t\t      <p><b>签到时间</b>：1小时前\t\t\t\t  </p>\t\t\t    </div>\t\t\t  </div>';
    $$('#mobel_table').html(ss);
}