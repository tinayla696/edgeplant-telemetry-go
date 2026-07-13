# 開発・運用ルール

## ブランチ規約

- `feature/*`
- `bugfix/*`
- `hotfix/*`
- `docs/*`
- `chore/*`

## コミット規約

- 形式: `type(scope): subject`
- 例: `feat(mq): add rabbitmq broker adapter`

## 運用ポイント

- Docs as Code: コード変更と同一PRで docs 更新
- `main` / `develop` への直接 push 禁止
- SIGTERM時は未送信処理を止めて安全終了
- GitHub Pages 公開は `main` ブランチ起点の GitHub Actions artifact デプロイで行い、`gh-pages` ブランチへ静的ファイルを直接コミットしない
