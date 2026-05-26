const GPTSDK = require('./gptsdk');

async function runTest() {
    // 实例化 SDK
    // 如果你有可用的代理，可以在这里配置（需取消内部 https-proxy-agent / undici 代码注释并安装相应包）
    // const sdk = new GPTSDK({ proxy: 'http://127.0.0.1:7890' });
    const sdk = new GPTSDK();

    console.log("====== 开始 GPTSDK 接口连通性测试 ======\n");

    // 注意：这里请填入你用于测试的真实 Refresh Token (RT) 或 Session Token (ST)
    // 如果填入空字符串或假数据，必然会报 400 或 401，但如果报 403 则说明被风控拦截了。
    const testRT = 'dummy_refresh_token';
    const testST = 'eyJhbGciOiJkaXIiLCJlbmMiOiJBMjU2R0NNIn0..3Sz8-JaOJT9iEX5q.Wzlkmpf384VzBV7GgQvQU7FmYH_pOsgXNFQRrvEdUbOdoBDcTuuqWHaFRv4WZIHCeRc8SnqeWMkXwxXPuoirdjhulADa5DuyNDPdO7ExQX3txdtcx1PvK92YntZ88Ft9XCEqpF2XpEVvUHJkubIAEiOxJZhiUAEP3opgBhr5CXCauIBgrKU7vSYTJqcJIzU72XNbBNrDhswCz2_WqOpjPrfSKTqp7IhFFTBGf0GiL0w4snfHdiNk89V-hQNaFCv3KELQ8NsIuQkjmehOM_4gUIhNDiNF93j5Xs9-d0uS7GYaiyYFkUB-JtYXZFug7lfR5CrRCvgZ3tcmB7LfDQ8R4f4COrOUj5S3b9ylB4sQxBdlu_Wd2Y30On_um-wcQxH424ABnIW2sJfaNCH8fp0_8lWKUZlRa3sQqJe8k9XrOFZ2oe7_JwT_QIopBSRShVaYFzbIXV-3HXgwFznVy35m5Ck6jJ85crEqdThPASeOgZ1JEDjEvUpqL1-vwWv2ri6u8TRY2K7EZFZ2gVjUXU_kVnEvrIMhtMwz1u_cMegZz60ZOz-Rvw-2SuMGLGnzMcEHmO-r1WeGfvnxOFxCNMzEF4f7muaBUTxT1sxR-shlHL-i0C5gqYeCB9yPeVkK0OMPOojSKdo_iArSbG9E1XIptx0eN9HxV1CRGqwlqv5OH0wXfWWFhPU2xch2A0a3uXdl92SWRxui9djpaTGqYFYJrKgGkH05y01qFHMwrSInIsvf7IK4enICMcCUzl712-WOq1LDcpQoNCOWtxxElG3Z9r5VsjQDeFvHv86gW7XxpsjkiDRckqwTRPpHwHgSVboMK1j1RUg43YfDpE3_9-Kqk2dgwp4ruDeDm-cx8JgkEJNnH3Q_wYIZBwJ6xHU-0mQwOdjKVN24sfZnvMVPdtf07foQaOVIfYYJptk-9vdyP6s8EkHO22LSyM0XZ4Sj4C_oor-yAJ0UCdvOrD7JmyyJfW5o4dPBbvzOE0sYz7qWfHx2C0CsxZwqEc8ZkBWC_LAjqTjGwVOKvjn6B0lssZLvJmLGHkRAo_0Wd6Idl_NwaSyHTE2j2TAFcjvD2PMCVpnc20wHZ_Gk1Uy11eD6DtyKPZ8GZ5-9vIxXjx5tZ02Fx20k0Z1NhCjXRI-XRFD_r_LehQOhtaiJ3hlZmwlTmIunh4fdV0M0e3dLH1p24bILfkkxGsXD-Et4IfozJNOh2_3qRikX3PE9QMeIMx94_9uRotzWSj074yr46KQPr9DG-1quPrrA9A-6_nMm_JUKooYiI4pB9-Y0mjoOLdtYObthUuY_I1m-eqQ1Y9Teo3frgCMEBtszcaV1HMIyIsrTj9vGuUKMVm5W7iV17LdW85v8nzmgZpk4xcoNhCrg4DNjEKV4KGNkG9oibsQj846-ok2PRlDFpEvX-uWmoIfHeyAR8UeuUYAMX-77V-LUFqfT027QDrVQQZ2cpf5QfTF5Iouhyvhg_auHX9xzxjy4y9l-EPJG1n0zlTP65ISGr9lrUSGPfQa0kn8hI1LkuqGU-8M34SjRB-byoAd0WGiVQwkbEe7pKjo27MO58bCQ5N2TVHZ1NziELgm1k_jOVlFUoaLgetjzGHYmKaLNCVkYEZytM6uovEIYSOgRDQ9Jby-SPYMM6NX5ZX3ciUba3lKJg8DzodBs6sd81vmzC4dTFPXIdLihU6yNFcadbmWgrI00aHw9LkyDqHS7xZc2S6JQBmupfq3zV64LcWif0bABgChLha40oMDwEvyPGQ-6wLJLRGPLvHEdafHbesqyGGN0GW-3ym5Yhd-u1viZ3YBLoLgv5youNeMnvtRyQGSF2297Heym0sYboiPJR-KPH8wJjNefKTk0CYtse93jXQaatisaq5hIKI2iduXAadEd7SkviaEC_mMZOvZhXwDYyQvDrad2TxOTH1jrEPFXYKx__QwHDaL9v-B81B5z6eIDSKnBntBGeBljc6tuK8ftX71q_WSrKFiFZCIUx8v3Z_xP_lvTgo5wBbFGCDghRK1MiPwJPHeYT5bOSpoePvojA7NwWdrYikmJ6dU_TKbqzMPHFhexK3emFwHuET-kxWwmQTuhT3wEOYF9cZRs4N2J6wDI6R5W5AqAUM5l8XQkS9KqoZBOwbJZTcJVyNoDoyhS-9uaU9qziGRpOhz0gmm8I4eytZFUO-mNbPT4F2zYx43VTl1bFjKOJNx0Wjvm0iWccqUztn4UCk2si8YN9_pIAPQGbzMD07E3jqu7J3yLTQZg2mJYmWYf61wRefGc3kEPy2XJw6ulQJT_DTUAXTplr6z1XiPq2D5kTF0vIfo8hxoEDpZ16VCNUgR_5wWZDmO2es32t5wJwWmo916B8MP9RPaG8WfQQQ3iXMMyXxTXO5dfsINtaPYHNRw-2BwWsiwYKzPIlghAoMKeAWqTE6qNr5GyZ4h1T9BRiGKsLlqCcQ9FzqMuIxtYOf3TkG2fxAduBAPvmN0eFu6EhKxVn126jNP7at8j4jQ7COjWSAYmiU3S9LeqkRyafaWhGR6N_9nIfhK2dbSmQ4U56KQ2ZASg7JS5X39kYnBvdcX4EFZ71XNZ4PkNOY-OXXlpfa1-Di3sRx82r9XYwmiselwYrm3yTxjowL5tanZqRFRlHW4KEp6i9EY_oGBLEwUULA4VcWoXjx1W2iIP8LctTyxaKcsDjZciMeN1KAPfsWQHG22JdtR8SUZVAtCddxU0HqFL0zl_apJzAfCVlCM7OHGXd3MRiuIbahZsacUvpAtT1tIZh4mdRI_kBOHBjZvApnKrnc5yCObt7ofzBrhfd4BfHboNgJki3c8hMyPv04bgNBWS3yx6Uxy0e2eLMSrkJgWxISqVE8SwaxHW7LqhD0TZUEG6lK5xerbYhexqzO1QaYibweKIs2ZIjs96AC2prRP5JKGa6x758afbtJH_R8smPCnagULluj2S0BHe5FAIWWgzG50ldB5EGzUr5Wy9nJ33bA-vX5p-Faq4-lK09EyWOIdCYDwboKM94JUVfyBVFPgP3cxSgfod-1LKYQrL2DAQ59IFpSUPm8a1gOXrZ5SGCNW-T_95Hmyu_C-zeNUU1_wDg1bdet8eENM-cGxAxq_05EWXjR0lWKxoCWiU5LPB3MYipdn9TbpLhP1Tmr7f3PxpWE7C81HN36oi3j-3tg-v4rxlOarsi8Dqt7wz466Nm2rnPH2YkbWu0dLeBoqN_mBxx-OtEvA0eMAwiSTx6no8eQiG-FJCkCEeAPPIia7ssLPGmPVFgDWPG1YvpSaIV5NzDz73jkpk6DGHmUzB4SuwxIV_jQx0QNBGTzuoYT-NF5OCpw9cmgUnbzeKY2gfgUG9VTXUvTl_ogh85_PR0pbYUc1gIir1WQnAOVs6KsuMQ52chxw02KiyKJgV3_vHTfCvnDB42g5oOzW5g8_-WoXRiX9lsf3rIT51wTifHdJjiHOQQ_kUirbtylmY88OKquSNTZ6DcY798cwJbCdsnD-p5GPaJFSbQKtHwETLwGHQrn_F6yK6W8lD5Z7w5xUpAtBLse7dyTEM1KFgzv3hMpbGLI3vBe3C2XW3SOQJdNYtiNvTzsAAzJZ9Mzchm0ZyhH0rzURkr10m3RPAfMfb9Zj3Bz7Cnt5CFAI9PkYdt1NuQ-BpeJgVTsxQvnovwGiJ0daOKkFyHQdsTmv3ZJ-igfs0YIf45NmpfJM_b9f_y1da1i7aUllFPvp_Ja8hwISMwa2GWjPOBzN3iu6cHX3w-wrXLS2_aSqviYoc6hkGflzCk8fZjvzXDYzIkA02ohLKYrreKigSjw6OdBJS4ynm4t_izatP2Sbkb40HbQt-0sQ4UBpvwcKPa6KPDkzt.EQL1QXs2hFY9u-Ii3r6eiA';

    // 1. 测试 rtToAT
    // try {
    //     console.log("[1] 测试 rtToAT 接口 (POST auth.openai.com/oauth/token)...");
    //     console.log("请注意观察报错：如果是 403 说明被 Cloudflare 盾拦截，如果是 400/401 说明网络通了只是 Token 不对。");
    //     const rtResult = await sdk.rtToAT(testRT);
    //     console.log("✅ rtToAT 成功:", rtResult);
    // } catch (e) {
    //     console.error("❌ rtToAT 失败:", e.message);
    // }

    console.log("\n----------------------------------------\n");

    // 2. 测试 stToAT
    try {
        console.log("[2] 测试 stToAT 接口 (GET chatgpt.com/api/auth/session)...");
        const stResult = await sdk.stToAT(testST);
        console.log("✅ stToAT 成功:", stResult);
    } catch (e) {
        console.error("❌ stToAT 失败:", e.message);
    }

    console.log("\n----------------------------------------\n");

    // 3. 测试 verifyATOnWeb (可选)
    // 如果你有一个有效的 Access Token，可以将它放在这里测试
    const testAT = 'eyJhbGciOiJSUzI1NiIsImtpZCI6IjE5MzQ0ZTY1LWJiYzktNDRkMS1hOWQwLWY5NTdiMDc5YmQwZSIsInR5cCI6IkpXVCJ9.eyJhdWQiOlsiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS92MSJdLCJjbGllbnRfaWQiOiJhcHBfWDh6WTZ2VzJwUTl0UjNkRTduSzFqTDVnSCIsImV4cCI6MTc4MDUwNDk0MiwiaHR0cHM6Ly9hcGkub3BlbmFpLmNvbS9hdXRoIjp7ImFtciI6WyJvdHAiLCJ1cm46b3BlbmFpOmFtcjpvdHBfZW1haWwiXSwiY2hhdGdwdF9hY2NvdW50X2lkIjoiN2M2OTdhMWUtNzVlNC00ZjEyLTg2ODMtM2FjMWE0MWIwNmQ5IiwiY2hhdGdwdF9hY2NvdW50X3VzZXJfaWQiOiJ1c2VyLUNzdHlLMTQyem9pejdQcFhTV29jNWE2VF9fN2M2OTdhMWUtNzVlNC00ZjEyLTg2ODMtM2FjMWE0MWIwNmQ5IiwiY2hhdGdwdF9jb21wdXRlX3Jlc2lkZW5jeSI6Im5vX2NvbnN0cmFpbnQiLCJjaGF0Z3B0X3BsYW5fdHlwZSI6InBsdXMiLCJjaGF0Z3B0X3VzZXJfaWQiOiJ1c2VyLUNzdHlLMTQyem9pejdQcFhTV29jNWE2VCIsInVzZXJfaWQiOiJ1c2VyLUNzdHlLMTQyem9pejdQcFhTV29jNWE2VCJ9LCJodHRwczovL2FwaS5vcGVuYWkuY29tL3Byb2ZpbGUiOnsiZW1haWwiOiJjaGVuZ3Rpbmdub3JlZGlAbWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZX0sImlhdCI6MTc3OTY0MDk0MSwiaXNzIjoiaHR0cHM6Ly9hdXRoLm9wZW5haS5jb20iLCJqdGkiOiI1MzUxOGY0NS1kMjZiLTQzODMtYmUxZi0yYzBjMDY4MWYxNTMiLCJuYmYiOjE3Nzk2NDA5NDEsInB3ZF9hdXRoX3RpbWUiOjE3Nzk2NDA5Mzg1NTEsInNjcCI6WyJvcGVuaWQiLCJlbWFpbCIsInByb2ZpbGUiLCJvZmZsaW5lX2FjY2VzcyIsIm1vZGVsLnJlcXVlc3QiLCJtb2RlbC5yZWFkIiwib3JnYW5pemF0aW9uLnJlYWQiLCJvcmdhbml6YXRpb24ud3JpdGUiXSwic2Vzc2lvbl9pZCI6ImF1dGhzZXNzX0l4VEtmYkhJTko0cUhFMnFVNWxaaEc4RyIsInNsIjp0cnVlLCJzdWIiOiJhdXRoMHxKWW5WOHZkdVQxemNEeXJ1U0ZKWEMybkYifQ.CCUrT0Sq4m_f6G_lbNFG4YgNET0IIlIW8LHzLIIpVDIn3xOVj9PQ1txWtHRT-E5zkn1WQ_LYCRzYQAheyDDBz6Y8dAycuAYJ1QbH1UATRMQ3WbKbS924OapoWkbNu16MpPK_PSqTi5N_SURlWTqvOcYHhFkoNoCAotUWjnyxcYrlJkkGZtHfJcmlbKWSWbX17wQ9Gf0aiTmjgU1ODAD4DSQ3tGhxGfeYwCk5L6B_PADAGCrbxhpEI-4TbJlyzbpO_Rq0UQvKtzKOSVbm8meI7W_XLpWc0zLc9loNHjqTb7x-VqAxIaKVxcq-u_c32NVBeNicaDKpCRibnDIA29uVYrlqO8Bve8fQZiSeQMfxDRG1H4lzkOlUmWntmF68O0oSMffGz8NxO_y3DFgvgxd2Pomt3ZWm7_9z00RN8kKcp54nKiB2apM-ddPF0AINRHwZY3FKQb1EVxA8WOHwQ2r7zq0zjFJaBUlVjC1zOCqbUE09JD5z6B5dpky805yQyfmoUUYEB7sEMmBm5x4EH3B6RLI0Tdg4ctmjqNgNkFDQezqdslumKQkTapD7itKVAA213jfL3SD_muSNbh2OcXzdXADrZaW9SAH9y0PMIJxGLOHzJfdoWLi8Ce4c8yOeMGarp2L9VhwIppZw3b8deIk_-p0oIeJ9FCFCA8Tqxft6uvI';
    if (testAT !== 'dummy_access_token') {
        try {
            console.log("[3] 测试 verifyATOnWeb 接口 (GET chatgpt.com/backend-api/me)...");
            const status = await sdk.verifyATOnWeb(testAT);
            console.log(`✅ verifyATOnWeb 响应状态码: ${status} ${status === 200 ? '(Token被Web接受)' : '(被拒绝)'}`);
        } catch (e) {
            console.error("❌ verifyATOnWeb 失败:", e.message);
        }
    } else {
        console.log("[3] verifyATOnWeb 测试已跳过（请填入真实的 Access Token 后测试）");
    }

    console.log("\n----------------------------------------\n");

    // 4. 测试 generateImage
    const testApiKey = 'sk-xxx'; // 替换为你的 gpt2api 服务端 API Key
    const testBaseUrl = 'http://127.0.0.1:8080/v1'; // 替换为你的 gpt2api 服务地址
    
    if (testApiKey !== 'sk-xxx') {
        try {
            console.log(`[4] 测试 generateImage 画图接口 (${testBaseUrl}/images/generations)...`);
            const imgResult = await sdk.generateImage("一只可爱的橘猫", testApiKey, testBaseUrl);
            console.log("✅ 画图成功:", JSON.stringify(imgResult, null, 2));
        } catch (e) {
            console.error("❌ 画图失败:", e.message);
        }
    } else {
        console.log("[4] generateImage 画图测试已跳过（请填入真实的 API Key 和 服务地址 后测试）");
    }

    console.log("\n====== 测试结束 ======");
}

runTest();
