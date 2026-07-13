# Public 運用・Runner・Release 方針

## Public リポジトリ運用方針

- 本リポジトリは Public 前提で運用する。
- ドキュメント公開は GitHub Pages を利用する。
- ドキュメント公開物は GitHub Actions artifact から配信し、`gh-pages` ブランチへの静的ファイル直接コミットは行わない。

## Pages 公開方針

- workflow: `Docs Publish to GitHub Pages`
- trigger: `main` への push または手動実行
- 公開 URL: `https://tinayla696.github.io/edgeplant-telemetry-go/`

### 公開確認チェックリスト

1. `Docs Publish to GitHub Pages` が `success` になる
2. ホームが表示される
3. MQTT 試験結果ページが表示される
4. RabbitMQ 試験結果ページが表示される

## self-hosted runner 最小構成（このリポジトリ用）

`Telemetry Validation CI` の `E2E VCAN Evidence` ジョブは `runs-on: [self-hosted, linux]` で実行されます。

### 必須要件

- Linux ホスト（Ubuntu 推奨）
- Docker / Docker Compose が使える
- runner ユーザーが Docker を実行できる（`docker` グループ等）
- GitHub repository runner として `tinayla696/edgeplant-telemetry-go` に登録済み
- runner labels に `self-hosted` と `linux` を含む

### 登録手順（最小）

1. GitHub リポジトリ画面から runner 登録コマンドを取得
: `Settings` -> `Actions` -> `Runners` -> `New self-hosted runner`

2. Linux ホストで実行

```bash
mkdir -p ~/actions-runner && cd ~/actions-runner
# 以下は GitHub 画面で表示される URL/バージョンを使用
curl -o actions-runner-linux-x64.tar.gz -L <RUNNER_TARBALL_URL>
tar xzf ./actions-runner-linux-x64.tar.gz
./config.sh --url https://github.com/tinayla696/edgeplant-telemetry-go --token <RUNNER_TOKEN> --labels self-hosted,linux
./run.sh
```

3. サービス化（任意だが推奨）

```bash
sudo ./svc.sh install
sudo ./svc.sh start
```

### runner 登録確認

```bash
gh api repos/tinayla696/edgeplant-telemetry-go/actions/runners --jq '{total_count:.total_count,runners:[.runners[]|{name,status,labels:[.labels[].name]}]}'
```

`total_count` が 1 以上になれば runner 側の準備は完了です。

## main の E2E 再実行（runner 登録後）

```bash
RUN_ID=$(gh run list -R tinayla696/edgeplant-telemetry-go --workflow "Telemetry Validation CI" --branch main --limit 1 --json databaseId --jq '.[0].databaseId')
gh run rerun "$RUN_ID" -R tinayla696/edgeplant-telemetry-go
```

### 再実行後の確認

1. `E2E VCAN Evidence (mqtt)` が `success`
2. `E2E VCAN Evidence (rabbitmq)` が `success`
3. artifacts が作成される
: `telemetry-e2e-mqtt-evidence`
: `telemetry-e2e-rabbitmq-evidence`

## main マージ時バイナリ Release 自動化

- workflow: `Telemetry Build and Release`
- trigger: `main` への push（PR マージを含む）
- 生成物:
  - `telemetry-linux-amd64`
  - `telemetry-linux-arm64`
  - `SHA256SUMS`
- 上記を GitHub Release に自動添付する
