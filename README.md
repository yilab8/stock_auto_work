
# KHAM Ticket Bot

此專案提供一個以 Python 標準函式庫建構的指令工具，用來協助與寬宏售票網頁（`UTK0101_03.aspx`）互動。功能重點如下：

- 解析頁面中的 HTML 表單並自動帶入欄位值。
- 使用 `urllib` 與 CookieJar 維持會話並執行登入流程。
- 透過 JSON 設定檔描述搶票流程（抓頁、送出表單、輪詢等待關鍵字）。
- 提供命令列工具：`login`、`dump-forms`、`run`，方便檢視表單與執行設定流程。
- 內建 Tkinter 圖形介面，讓常用操作可以透過按鈕完成。

## 環境需求

- Python 3.11（本專案僅使用標準函式庫，無需安裝額外套件）。

## 安裝與測試

使用 `pip` 安裝後會自動註冊 `ticketbot` 與 `ticketbot-gui` 指令，方便在正式環境中直接呼叫 CLI 或桌面版工具：

```bash
pip install .
ticketbot --help
ticketbot-gui
```

若要確認功能或在開發階段進行驗證，可執行隨附的單元測試：

```bash
python -m unittest discover -s tests
```

所有測試均為離線模擬，確保表單解析、登入流程及 CLI 邏輯皆能正常運作。

## 使用方式

### 1. 查看表單結構

```bash
ticketbot dump-forms --url https://kham.com.tw/application/utk01/UTK0101_03.aspx
```

會列出頁面上偵測到的所有表單、欄位名稱與預設值，協助了解要覆寫的欄位。

### 2. 嘗試登入

```bash
ticketbot login \
  --account L125097509 \
  --password Aa@@850302 \
  --extra __EVENTTARGET= __EVENTARGUMENT= ctl00$ContentPlaceHolder1$btnLogin=登入
```

> 注意：目前環境對外網路受限，登入實際會連線失敗。若在有網路的環境執行，請確保帳密安全並遵守網站使用條款。

### 3. 依設定檔自動執行

先複製範例設定檔再依需求調整（例如覆寫欄位、輪詢條件）：

```bash
cp config.example.json my_config.json
```

將 `my_config.json` 中的 `${KHAM_ACCOUNT}`、`${KHAM_PASSWORD}` 改為環境變數或直接填寫帳密（建議使用環境變數）。接著執行：

```bash
export KHAM_ACCOUNT=L125097509
export KHAM_PASSWORD=Aa@@850302
ticketbot run --config my_config.json
```

`steps` 內的每個動作會依序執行：

- `type: "fetch"` 代表單純抓取頁面。
- `type: "submit"` 會自動從頁面挑選符合條件的表單並送出。`overrides` 可指定要覆寫的欄位值。
- `polling` 可設定輪詢頁面直到 HTML 內容出現指定關鍵字（例如座位開放）。

### 4. 使用圖形介面快速操作

除了命令列工具外，也可以啟動 Tkinter 製作的圖形介面。安裝完成後執行：

```bash
ticketbot-gui
```

會開啟一個視窗，提供登入、擷取表單與執行設定檔的按鈕與輸入欄位。所有輸出會顯示在視窗下方的輸出區域，方便直接點選操作。

## 注意事項

- 由於評測環境無法連線外部網路，無法實際驗證登入或購票流程；程式碼透過單元測試模擬 HTTP 回應來驗證邏輯。
- 搶票時請留意網站之服務條款與使用規範，避免違反相關規定。
- 若頁面表單欄位調整，可透過 `dump-forms` 重新檢視欄位並更新設定檔。

