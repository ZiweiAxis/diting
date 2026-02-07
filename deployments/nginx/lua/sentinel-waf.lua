-- Sentinel-AI WAF Lua 脚本
-- 与 OpenResty 配合使用

local cjson = require "cjson.safe"
local http = require "resty.http"

-- 共享内存
local decision_cache = ngx.shared.decision_cache
local locks = ngx.shared.locks

-- 常量
local SENTINEL_API_URL = "http://sentinel-api:8000"
local TIMEOUT = 2000  -- 2 秒
local CACHE_TTL = 300  -- 5 分钟

-- 初始化
function init()
    ngx.log(ngx.NOTICE, "Sentinel-AI WAF Lua script initialized")
end

-- 调用 Sentinel-AI API 分析
function call_sentinel_api(req)
    local httpc = http.new()
    httpc:set_timeout(TIMEOUT / 1000)
    
    local res, err = httpc:request_uri(
        SENTINEL_API_URL .. "/analyze",
        {
            method = "POST",
            body = cjson.encode(req),
            headers = {
                ["Content-Type"] = "application/json",
                ["X-Sentinel-WAF"] = "nginx"
            },
            keepalive = true
        }
    )
    
    if not res then
        ngx.log(ngx.ERR, "Sentinel-AI API error: ", err)
        return nil, err
    end
    
    if res.status ~= 200 then
        ngx.log(ngx.WARN, "Sentinel-AI API returned: ", res.status)
        return nil, "API returned " .. res.status
    end
    
    local decision, err = cjson.decode(res.body)
    if not decision then
        ngx.log(ngx.ERR, "Failed to decode response: ", err)
        return nil, err
    end
    
    return decision, nil
end

-- 缓存决策
function cache_decision(cache_key, decision)
    decision_cache:set(cache_key, decision.action, CACHE_TTL)
end

-- 获取缓存
function get_cached(cache_key)
    return decision_cache:get(cache_key)
end

-- 主分析函数
function analyze_request()
    local method = ngx.var.request_method
    local uri = ngx.var.request_uri
    local host = ngx.var.host or ngx.var.http_host
    local remote_addr = ngx.var.remote_addr
    local user_agent = ngx.var.http_user_agent or "-"
    
    -- 读取请求体
    ngx.req.read_body()
    local body = ngx.var.request_body or ""
    
    -- 生成缓存键
    local cache_key = method .. ":" .. uri .. ":" .. remote_addr
    
    -- 检查缓存
    local cached = get_cached(cache_key)
    if cached then
        ngx.log(ngx.INFO, "Using cached decision: ", cached)
        return cached
    end
    
    -- 准备分析请求
    local headers = {}
    for k, v in pairs(ngx.req.get_headers()) do
        headers[k] = v
    end
    
    local analyze_req = {
        method = method,
        uri = uri,
        headers = headers,
        body = body:sub(1, 1000),  -- 限制大小
        client_ip = remote_addr,
        host = host,
        timestamp = ngx.now() * 1000
    }
    
    -- 调用 Sentinel-AI API
    local decision, err = call_sentinel_api(analyze_req)
    
    if not decision then
        -- 出错时默认放行
        ngx.log(ngx.WARN, "Sentinel-AI API failed, allowing request")
        return "ALLOW"
    end
    
    -- 缓存决策
    cache_decision(cache_key, decision)
    
    return decision.action
end

-- 执行决策
function execute_decision(decision)
    if decision == "BLOCK" then
        ngx.status = 403
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            error = "Request blocked by Sentinel-AI WAF",
            message = "This action is not allowed"
            code = "FORBIDDEN"
        }))
        ngx.exit(403)
    elseif decision == "REVIEW" then
        ngx.status = 202
        ngx.header["Content-Type"] = "application/json"
        ngx.say(cjson.encode({
            message = "Request pending approval",
            code = "PENDING"
        }))
        ngx.exit(202)
    else
        -- ALLOW: 继续到 proxy_pass
        ngx.header["X-Sentinel-Protected"] = "true"
        ngx.header["X-Sentinel-WAF"] = "v2.0"
    end
end

return _M
