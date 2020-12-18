var $$ = mdui.JQ;
$$(function () {
    init("");
});
function init(key){
    $$.ajax({
        method: 'GET',
        url: 'data/sign.json',
        success: function (data) {
            data = eval('(' + data + ')');
            $$("#lb_uc").html("用户数<span> "+data.tuc+"</span>");
            $$("#lb_tc").html("贴吧数<span> "+data.ttc+"</span>");
            $$("#lb_sc").html("已签到<span> "+data.tsc+"</span>");
            $$("#lb_vc").html("Cookie失效<span> "+data.tvc+"</span>");
            $$("#lb_bec").html("异常或黑名单<span> "+data.tbec+"</span>");
            $$("#lb_support").html("名人堂助攻<span> "+data.tsuc+"</span>");
            if((navigator.userAgent.match(/(phone|pad|pod|iPhone|iPod|ios|iPad|Android|Mobile|BlackBerry|IEMobile|MQQBrowser|JUC|Fennec|wOSBrowser|BrowserNG|WebOS|Symbian|Windows Phone)/i))) {
                $$('#mobel_table').show();
                $$('#pc_table').hide();
                var htmStr = ""
                $$.each(data.Sts, function (i, item) {
                    if(key != "" && item.name.indexOf(key) == -1) {
                        return true;
                    }
                    var html = "<div class=\"mdui-panel-item\">"+
                        "                <div class=\"mdui-panel-item-header\">"+
                        "                    <div class=\"mdui-panel-item-title\">"+
                        "                        <div class=\"mdui-chip\">"+
                        "                            <img class=\"mdui-chip-icon\" src=\""+item.head_url+"\"/>"+
                        "                            <span class=\"mdui-chip-title\">"+
                        "                            </span>"+
                        "                        </div>"+
                        "                    </div>"+
                        "                    <div class=\"mdui-panel-item-summary\">"+getDateDiff(item.sign_time)+"</div>"+
                        "                    <div class=\"mdui-panel-item-summary\">"+
                        "                        <i class=\"mdui-icon material-icons mdui-text-color-green\">check_box</i>"+
                        "                   </div>"+
                        "                <i class=\"mdui-panel-item-arrow mdui-icon material-icons\">keyboard_arrow_down</i>"+
                        "            </div>"+
                        "            <div class=\"mdui-panel-item-body\">";
                    if(!item.is_valid){
                        html+="                <p class=\"mdui-text-color-red-900\"><b>ID</b>: "+item.name+"</p>";
                    }else{
                        html+="                <p class=\"mdui-text-color-teal-900\"><b>ID</b>: "+item.name+"</p>";
                    }
                    html += "                <p class=\"mdui-text-color-blue-900\"><b>签到</b>: "+(item.signed+item.bq)+"/"+item.total+"</p>"+
                        "                <p class=\"mdui-text-color-red-900\"><b>异常或黑名单</b>: "+(item.excep+item.black)+"</p>"+
                        "                <p class=\"mdui-text-color-blue-900\"><b>名人堂助攻</b>: "+item.support+"</p>"+
                        "                <p><b>文库</b>:"+item.wenku+"</p>"+
                        "                <p><b>知道</b>:"+item.zhidao+"</p>"+
                        "                <p><b>最近一次签到</b>: "+getDateDiff(item.sign_time)+"</p>"+
                        "                <p><b>耗时</b>: "+getDuration(item.timespan)+"</p>"+
                        "            </div>"+
                        "        </div>";
                    htmStr+=html;
                });
                $$('#mobel_table').html(htmStr);
            }else{
                $$('#mobel_table').hide();
                $$('#pc_table').show();
                var htmStr = ""
                $$.each(data.Sts, function (i, item) {
                    if(key != "" && item.name.indexOf(key) == -1) {
                        return true;
                    }
                    var html =" <tr>"+
                        "            <td>"+
                        "                <div class=\"mdui-chip\">"+
                        "                    <img class=\"mdui-chip-icon\" src=\""+item.head_url+"\"/>"+
                        "                    <span class=\"mdui-chip-title <#if bean.cookie_valid ==0>mdui-text-color-red</#if>\">"+
                        item.name+
                        "                    </span>"+
                        "                </div>"+
                        "            </td>"+
                        "            <td><div class=\"mdui-text-color-blue-900\">"+(item.signed+item.bq)+"/"+item.total+"</div></td>"+
                        "            <td>"+
                        "                <div class=\"mdui-text-color-red-900\">"+(item.excep+item.black)+"</div></td>"+
                        "            <td>"+
                        "                <div class=\"mdui-text-color-red-900\">"+(item.support)+"</div></td>"+
                        "            <td>"+
                        "                <div class=\"mdui-text-color-black-900\">"+item.zhidao+"</div></td>"+
                        "            <td>"+
                        "                <div class=\"mdui-text-color-black-900\">"+item.wenku+"</div></td>"+
                        "            <td>"+getDateDiff(item.sign_time)+"</td>"+
                        "            <td>"+getDuration(item.timespan)+"</td>"+
                        "        </tr>";
                    htmStr+=html;
                });
                $$('#pc_body').html(htmStr);
            }
        }
    });
    mdui.mutation();
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
        init(key);
    }
});