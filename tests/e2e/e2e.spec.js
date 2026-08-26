/**
 * E2E 端到端系統整合與全功能驗證測試
 * 測試範圍：前端 Web 服務、後端 API 各模組與資料庫真實連線
 */
const http = require('http');
const fixtures = require('./test_fixtures.json');

const HEADERS = {
  'Authorization': fixtures.auth.token,
  'X-Mock-Role': fixtures.auth.role,
  'X-Mock-User-ID': fixtures.auth.userId,
  'Content-Type': 'application/json'
};

function request(url, options = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(url);
    const reqOptions = {
      hostname: parsed.hostname,
      port: parsed.port,
      path: parsed.pathname + parsed.search,
      method: options.method || 'GET',
      headers: { ...HEADERS, ...(options.headers || {}) }
    };

    const req = http.request(reqOptions, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        try {
          const parsedBody = body ? JSON.parse(body) : null;
          resolve({ status: res.statusCode, headers: res.headers, data: parsedBody });
        } catch (e) {
          resolve({ status: res.statusCode, headers: res.headers, raw: body });
        }
      });
    });

    req.on('error', reject);
    if (options.body) {
      req.write(typeof options.body === 'string' ? options.body : JSON.stringify(options.body));
    }
    req.end();
  });
}

async function runE2ESuite() {
  console.log('================================================================');
  console.log('🚀 開始執行 LTC 長照系統 E2E 全功能架構整合測試 (tests/e2e)');
  console.log('================================================================\n');

  const { apiBase, webBase, testEntities } = fixtures;

  const testCases = [
    // 1. 前端頁面容器連線
    { name: '前端首頁 Web (Port 3000)', url: `${webBase}/`, method: 'GET', expectedStatus: [200, 304] },
    
    // 2. 健康檢查
    { name: '後端健康檢查 (GET /api/health)', url: 'http://localhost:8080/api/health', method: 'GET', expectedStatus: [200] },

    // 3. 總覽儀表板
    { name: '儀表板統計 (GET /dashboard/stats)', url: `${apiBase}/dashboard/stats`, method: 'GET', expectedStatus: [200] },
    { name: '儀表板指標 (GET /dashboard/metrics)', url: `${apiBase}/dashboard/metrics`, method: 'GET', expectedStatus: [200] },

    // 4. 個案主檔與排班
    { name: '個案清單查詢 (GET /cases)', url: `${apiBase}/cases?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '個案明細查詢 (GET /cases/:id)', url: `${apiBase}/cases/${testEntities.caseId}`, method: 'GET', expectedStatus: [200] },
    { name: '個案身分證解密 (POST /cases/:id/reveal)', url: `${apiBase}/cases/${testEntities.caseId}/reveal`, method: 'POST', expectedStatus: [200] },
    { name: '個案排班查詢 (GET /cases/:id/schedule)', url: `${apiBase}/cases/${testEntities.caseId}/schedule`, method: 'GET', expectedStatus: [200] },

    // 5. 主檔管理（據點、車輛、司機）
    { name: '據點清單查詢 (GET /sites)', url: `${apiBase}/sites?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '車輛清單查詢 (GET /vehicles)', url: `${apiBase}/vehicles?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '司機清單查詢 (GET /drivers)', url: `${apiBase}/drivers?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '司機身分證解密 (POST /drivers/:id/reveal)', url: `${apiBase}/drivers/${testEntities.driverId}/reveal`, method: 'POST', expectedStatus: [200] },

    // 6. 表單同步與欄位對應
    { name: '表單同步清單 (GET /forms)', url: `${apiBase}/forms`, method: 'GET', expectedStatus: [200] },
    { name: '手動同步表單 (POST /forms/:id/sync)', url: `${apiBase}/forms/${testEntities.formId}/sync`, method: 'POST', expectedStatus: [200] },
    { name: '待對應欄位清單 (GET /forms/columns)', url: `${apiBase}/forms/columns`, method: 'GET', expectedStatus: [200] },
    { name: '單筆欄位對應更新 (PATCH /forms/columns/:id/mapping)', url: `${apiBase}/forms/columns/${testEntities.columnId}/mapping`, method: 'PATCH', body: { mappingStatus: 'mapped', caseId: testEntities.caseId, legSeq: 1 }, expectedStatus: [200] },

    // 7. 搭乘月曆與異常處理
    { name: '搭乘月曆矩陣 (GET /rides/calendar)', url: `${apiBase}/rides/calendar?month=115-07`, method: 'GET', expectedStatus: [200] },
    { name: '異常搭乘清單 (GET /rides/issues)', url: `${apiBase}/rides/issues?issueType=conflict`, method: 'GET', expectedStatus: [200] },
    { name: '衝突裁決解決 (POST /rides/:id/resolve-conflict)', url: `${apiBase}/rides/ride_conflict_1/resolve-conflict`, method: 'POST', body: { vehicleId: testEntities.vehicleId, driverId: testEntities.driverId }, expectedStatus: [200] },
    { name: '未回報搭乘清單 (GET /rides/missing)', url: `${apiBase}/rides/missing?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },

    // 8. 營運報表
    { name: '車輛趟數表 (GET /reports/trip-summary)', url: `${apiBase}/reports/trip-summary?periodYm=115-07`, method: 'GET', expectedStatus: [200] },
    { name: '新竹接送時刻表 (GET /reports/hsinchu-schedule)', url: `${apiBase}/reports/hsinchu-schedule?date=2026-07-10`, method: 'GET', expectedStatus: [200] },

    // 9. 維修保養、司機出勤、車輛油資
    { name: '車輛保養清單 (GET /vehicles/maintenance)', url: `${apiBase}/vehicles/maintenance?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '司機月出勤紀錄 (GET /attendance)', url: `${apiBase}/attendance?month=115-07`, method: 'GET', expectedStatus: [200] },
    { name: '車輛油資紀錄 (GET /fuel-logs)', url: `${apiBase}/fuel-logs?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },

    // 10. 通知設定與催報任務
    { name: '通知收件人清單 (GET /settings/notification-recipients)', url: `${apiBase}/settings/notification-recipients`, method: 'GET', expectedStatus: [200] },
    { name: '通知歷史日誌 (GET /notifications/logs)', url: `${apiBase}/notifications/logs?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] },
    { name: '手動催報任務 (POST /tasks/check-missing-reports)', url: `${apiBase}/tasks/check-missing-reports`, method: 'POST', expectedStatus: [200] },

    // 11. 政府申報匯出與前置檢核
    { name: '匯出前置檢核 (POST /exports/precheck)', url: `${apiBase}/exports/precheck`, method: 'POST', body: { periodYm: '115-07', region: 'hsinchu' }, expectedStatus: [200] },
    { name: '匯出工作清單 (GET /exports)', url: `${apiBase}/exports?page=1&pageSize=10`, method: 'GET', expectedStatus: [200] },

    // 12. 稽核紀錄
    { name: '系統稽核紀錄 (GET /audit)', url: `${apiBase}/audit?page=1&pageSize=20`, method: 'GET', expectedStatus: [200] }
  ];

  let passed = 0;
  let failed = 0;

  for (const tc of testCases) {
    try {
      const res = await request(tc.url, { method: tc.method, body: tc.body });
      if (tc.expectedStatus.includes(res.status)) {
        console.log(`✅ [PASS] ${tc.name} -> HTTP ${res.status}`);
        passed++;
      } else {
        console.error(`❌ [FAIL] ${tc.name} -> 預期 [${tc.expectedStatus.join(', ')}]，實際收到 HTTP ${res.status}:`, res.data || res.raw);
        failed++;
      }
    } catch (err) {
      console.error(`❌ [ERROR] ${tc.name} -> 連線異常: ${err.message}`);
      failed++;
    }
  }

  console.log('\n================================================================');
  console.log(`📊 驗證結果彙整：總計 ${testCases.length} 項測試，通過: ${passed}，失敗: ${failed}`);
  console.log('================================================================');

  if (failed > 0) {
    process.exit(1);
  }
}

runE2ESuite();
