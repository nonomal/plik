---
layout: home

hero:
  name: Plik
  text: Temporary File Upload System
  tagline: A scalable & friendly temporary file sharing platform — like WeTransfer, self-hosted.
  actions:
    - theme: brand
      text: Documentation
      link: /guide/installation
    - theme: alt
      text: View on GitHub
      link: https://github.com/root-gg/plik

features:
  - icon: 🚀
    title: Powerful CLI
    details: Cross-platform Go client with archive, encryption, and auto-update support.
    link: /features/cli-client
  - icon: 🌐
    title: Modern Web UI
    details: Clean Vue 3 interface for uploading, downloading, and managing files.
    link: /features/web-ui
  - icon: 💾
    title: Multiple Backends
    details: File, S3, OpenStack Swift, Google Cloud Storage for data. SQLite, PostgreSQL, MySQL for metadata.
    link: /backends/data
  - icon: 🔒
    title: Security First
    details: Password protection, OneShot downloads, end-to-end encryption (age), XSRF protection.
    link: /guide/security
  - icon: 🔑
    title: Flexible Authentication
    details: Local accounts, Google, OVH, and OpenID Connect (OIDC) providers.
    link: /features/authentication
  - icon: ⚡
    title: Stream Mode
    details: Stream files directly from uploader to downloader — nothing stored on the server.
    link: /features/streaming
---
