#!/usr/bin/env bash
export curDir="`dirname $0`"
#curl -d '{"pay_amount":1, "order_time":"2018-10-28 12:00:00", "app_id":"1", "sign":"3e03ebf6a848b27a75f98f90c0225829", "channel_id":"demo", "sign_type":"MD5"}' "http://127.0.0.1:18080/v1/pay/test"

nowDate="$(date +'%Y-%m-%d %H:%M:%S')"
now="$(date +'%Y%m%d%H%M%S')"
orderId="${now}`date +%s%N | awk '{print substr($0,10,9)}'`$RANDOM"
channelId="isp"
method="WEB"
product_name="apple"
product_describe="apple"
user_ip="127.0.0.1"
#ext_json='{\"service_type\":\"jdpay_scan\"}'
ext_json='{\"service_type\":\"direct_pay\"}'

json='{"pay_amount":1, "out_trade_no": "'${orderId}'", "order_time":"'${nowDate}'", "app_id":"1", "channel_id":"'${channelId}'", "sign_type":"RSA", "method":"'${method}'", "product_name":"'$product_name'", "product_describe":"'$product_describe'", "user_ip":"'$user_ip'", "ext_json":"'$ext_json'"}'
echo "json==$json"
echo `$curDir/sign -j "$json" -d true` > $curDir/${orderId}.tmp
sign="`cat $curDir/${orderId}.tmp | awk '{print $NF}'`"
echo "sign===$sign"
json='{"pay_amount":1, "out_trade_no": "'${orderId}'", "order_time":"'${nowDate}'", "app_id":"1", "channel_id":"'${channelId}'", "sign_type":"RSA", "method":"'${method}'", "product_name":"'$product_name'", "product_describe":"'$product_describe'", "user_ip":"'$user_ip'", "ext_json":"'$ext_json'", "sign":"'$sign'"}'
echo "final json: $json"
curl -d "$json" "http://127.0.0.1:18080/v1/pay/${method}"