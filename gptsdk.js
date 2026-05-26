/**
 * GPTSDK - Node.js implementation for ChatGPT authentication flows
 * 
 * 注意：在实际生产环境中，由于 Cloudflare 的严格防护 (403 Forbidden)，
 * 原生的 Node.js fetch 或 axios 的 TLS 指纹极易被识别并拦截。
 * 建议配合 `https-proxy-agent` 使用干净的住宅代理，或者使用支持伪造 TLS 指纹的库（如 node-tls-client / got-scraping）。
 */

class GPTSDK {
    constructor(options = {}) {
        // 默认的 iOS 客户端 ID (如果代码里没传的话)
        this.defaultClientId = options.clientId || 'pdlLIX2Y72MIl2rhLhTE9VV9bN905kBh';
        this.proxy = options.proxy || null;
    }

    /**
     * 内部通用的请求封装
     */
    async _request(url, options = {}) {
        // 如果配置了代理，在 Node.js 18+ 原生 fetch 中可以通过 dispatcher 传入 (需结合 undici)
        // 或者使用 node-fetch 配合 https-proxy-agent。这里使用原生 fetch 演示标准用法。
        /*
        if (this.proxy) {
            const { ProxyAgent } = require('undici');
            options.dispatcher = new ProxyAgent(this.proxy);
        }
        */

        const response = await fetch(url, options);
        const data = await response.text();
        
        if (!response.ok) {
            let errMsg = `HTTP ${response.status}: `;
            errMsg += data.length > 200 ? data.substring(0, 200) + '...' : data;
            
            const err = new Error(errMsg);
            err.status = response.status;
            throw err;
        }
        
        try {
            const parsed = JSON.parse(data);
            return parsed;
        } catch (e) {
            return data;
        }
    }

    /**
     * 1. rtToAT: 使用 Refresh Token (RT) 获取新的 Access Token (AT)
     * POST https://auth.openai.com/oauth/token
     */
    async rtToAT(refreshToken, clientId) {
        const body = {
            client_id: clientId || this.defaultClientId,
            grant_type: 'refresh_token',
            redirect_uri: 'com.openai.chat://auth0.openai.com/ios/com.openai.chat/callback',
            refresh_token: refreshToken
        };

        const headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            // 保持和 iOS App 一致的 User-Agent
            'User-Agent': 'ChatGPT/1.2025.122 (iOS 18.2; iPhone15,2; build 15096)'
        };

        const res = await this._request('https://auth.openai.com/oauth/token', {
            method: 'POST',
            headers,
            body: JSON.stringify(body)
        });

        if (!res.access_token) {
            throw new Error('rt refresh: missing access_token in response');
        }

        let expAt;
        if (res.expires_in) {
            expAt = new Date(Date.now() + res.expires_in * 1000);
        } else {
            expAt = this._parseJWTExp(res.access_token);
        }

        return {
            accessToken: res.access_token,
            refreshToken: res.refresh_token,
            expiresAt: expAt
        };
    }

    /**
     * 2. stToAT: 使用 Session Token (ST) 获取新的 Access Token (AT)
     * GET https://chatgpt.com/api/auth/session
     */
    async stToAT(sessionToken) {
        const headers = {
            'Accept': 'application/json',
            'Referer': 'https://chatgpt.com/',
            // Web 端的请求需要真实的浏览器 User-Agent
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36',
            'Cookie': `__Secure-next-auth.session-token=${sessionToken}`
        };

        const res = await this._request('https://chatgpt.com/api/auth/session', {
            method: 'GET',
            headers
        });

        // 可能是空字符串或者空对象 `{}`
        if (!res || (typeof res === 'object' && Object.keys(res).length === 0)) {
            throw new Error('ST is expired or invalid, response is empty');
        }

        if (!res.accessToken) {
            throw new Error('Response missing accessToken field');
        }

        let expAt;
        if (res.expires) {
            expAt = new Date(res.expires);
        }
        if (!expAt || isNaN(expAt.getTime())) {
            expAt = this._parseJWTExp(res.accessToken);
        }

        return {
            accessToken: res.accessToken,
            expiresAt: expAt
        };
    }

    /**
     * 3. verifyATOnWeb: 校验 Access Token 是否能被 Web 端 (chatgpt.com) 接受
     * GET https://chatgpt.com/backend-api/me
     */
    async verifyATOnWeb(accessToken) {
        const headers = {
            'Authorization': `Bearer ${accessToken}`,
            'Accept': 'application/json',
            'Referer': 'https://chatgpt.com/',
            'Origin': 'https://chatgpt.com',
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
        };

        try {
            const response = await fetch('https://chatgpt.com/backend-api/me', {
                method: 'GET',
                headers
            });
            // 200 说明作用域兼容，401 说明不兼容（比如 iOS 的 RT 刷出来的 AT 可能在 Web 端报 401）
            return response.status;
        } catch (error) {
            throw new Error(`Network error during verify: ${error.message}`);
        }
    }

    /**
     * 4. generateImage: 调用标准画图接口 (OpenAI 兼容)
     * POST /images/generations
     * 
     * 提示：如果要直接调 chatgpt.com 网页版底层的 f/conversation 接口出图，
     * 会涉及复杂的 PoW 算力校验 (chat-requirements) 和 SSE 流解析，Node.js 单文件难以完美实现。
     * 因此强烈建议将此请求发给你部署好的 `gpt2api` 服务端，由 Go 后端去处理底层协议。
     * 
     * @param {string} prompt 提示词
     * @param {string} apiKey 你的 API Key (比如 gpt2api 签发的 sk-xxx)
     * @param {string} baseUrl API Base URL，例如 'http://127.0.0.1:8080/v1'
     * @param {Object} options 其他可选参数，如 model, size, n
     */
    async generateImage(prompt, apiKey, baseUrl = 'http://127.0.0.1:8080/v1', options = {}) {
        const url = `${baseUrl.replace(/\/$/, '')}/images/generations`;
        
        const body = {
            model: options.model || 'gpt-image-2', // gpt2api 的默认图像模型
            prompt: prompt,
            n: options.n || 1,
            size: options.size || '1024x1024',
            ...options
        };

        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${apiKey}`,
            'Accept': 'application/json'
        };

        try {
            const response = await fetch(url, {
                method: 'POST',
                headers,
                body: JSON.stringify(body)
            });
            
            const data = await response.text();
            
            if (!response.ok) {
                let errMsg = `HTTP ${response.status}: `;
                errMsg += data.length > 200 ? data.substring(0, 200) + '...' : data;
                throw new Error(errMsg);
            }
            
            return JSON.parse(data);
        } catch (error) {
            throw new Error(`Image generation failed: ${error.message}`);
        }
    }

    /**
     * 工具方法：解析 JWT 提取过期时间 (秒级 exp)
     */
    _parseJWTExp(token) {
        try {
            const parts = token.split('.');
            if (parts.length < 2) return new Date(Date.now() + 24 * 60 * 60 * 1000); // 兜底 +24h
            
            // base64url decode
            const base64Url = parts[1];
            const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
            const payloadStr = Buffer.from(base64, 'base64').toString('utf-8');
            const payload = JSON.parse(payloadStr);
            
            if (payload && payload.exp) {
                return new Date(payload.exp * 1000);
            }
        } catch (e) {
            // 如果解析失败走兜底逻辑
        }
        return new Date(Date.now() + 24 * 60 * 60 * 1000);
    }
}

module.exports = GPTSDK;
