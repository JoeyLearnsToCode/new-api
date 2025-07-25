# 渠道而外设置说明

该配置用于设置一些额外的渠道参数，可以通过 JSON 对象进行配置。主要包含以下几个设置项：

1. force_format
    - 用于标识是否对数据进行强制格式化为 OpenAI 格式
    - 类型为布尔值，设置为 true 时启用强制格式化

2. proxy
    - 用于配置网络代理
    - 类型为字符串，填写代理地址（例如 socks5 协议的代理地址）

3. thinking_to_content
   - 用于标识是否将思考内容`reasoning_content`转换为`<think>`标签拼接到内容中返回
   - 类型为布尔值，设置为 true 时启用思考内容转换

4. insecure_skip_verify
   - 用于标识是否跳过HTTPS证书验证（相当于curl的-k选项）
   - 类型为布尔值，设置为 true 时不严格校验HTTPS证书
   - 适用于自签名证书或证书过期的场景

--------------------------------------------------------------

## JSON 格式示例

以下是一个示例配置，启用强制格式化、思考内容转换和不严格HTTPS验证，并设置了代理地址：

```json
{
    "force_format": true,
    "thinking_to_content": true,
    "insecure_skip_verify": true,
    "proxy": "socks5://xxxxxxx"
}
```

--------------------------------------------------------------

通过调整上述 JSON 配置中的值，可以灵活控制渠道的额外行为，比如是否进行格式化以及使用特定的网络代理。
