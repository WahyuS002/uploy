# PRD — Authentication & Authorization for Multi-User PaaS

## 1. Context

Produk ini ditujukan untuk dipakai oleh user, bukan hanya admin internal. Karena itu, sistem auth tidak boleh lagi didesain sebagai “panel tertutup untuk satu admin”, melainkan sebagai fondasi untuk produk PaaS multi-user.

Versi sebelumnya masih berangkat dari asumsi bahwa hanya ada satu admin yang login ke dashboard. Pendekatan itu cukup untuk eksperimen awal, tetapi tidak cocok untuk produk yang nantinya dipakai banyak user dengan resource masing-masing.

Masalah utama yang harus diselesaikan bukan hanya “siapa yang boleh login”, tetapi juga:

- siapa yang memiliki workspace
- siapa yang boleh membuat project
- siapa yang boleh deploy
- siapa yang boleh melihat logs
- siapa yang boleh mengelola server, domain, dan environment variables

Dengan kata lain, kebutuhan sistem bukan hanya authentication, tetapi juga authorization berbasis tenant/workspace.

---

## 2. Objective

Membangun fondasi authentication dan authorization yang:

- cocok untuk produk PaaS multi-user
- sederhana dan realistis untuk fase awal
- tidak over-engineered
- memakai dependency seminimal mungkin
- tetap punya jalur upgrade ke social login dan arsitektur auth yang lebih kompleks di masa depan

---

## 3. Product Decision

### 3.1 Authentication strategy

Untuk fase awal, authentication akan dibangun langsung di app menggunakan:

- email/password login
- session berbasis cookie
- password hashing modern
- register/login flow sederhana

Pada fase berikutnya, sistem akan ditambah dengan:

- Sign in with GitHub
- Sign in with Google

Untuk saat ini, sistem **tidak** akan menggunakan:

- Authentik
- external identity provider terpisah
- SSO lintas aplikasi
- full OIDC architecture sebagai provider internal

Keputusan ini diambil agar tim fokus pada validasi inti produk PaaS, bukan terlalu cepat membangun identity platform sendiri.

---

### 3.2 Authorization strategy

Authorization akan didesain sejak awal sebagai **workspace-based authorization**, bukan admin-only authorization.

Artinya:

- user mendaftar sebagai user biasa
- setiap user memiliki workspace
- setiap resource penting terikat ke workspace
- akses ditentukan berdasarkan membership dan role di workspace tersebut

Role admin internal platform tetap boleh ada, tetapi itu dipisahkan dari role user produk.

---

## 4. Scope

## In Scope

### Phase 1

- user registration dengan email/password
- user login dengan email/password
- session management
- logout
- user profile dasar
- auto-create workspace saat registrasi
- user menjadi owner workspace pertamanya
- authorization dasar berbasis workspace membership
- pemisahan platform role dan workspace role

### Phase 2

- Sign in with GitHub
- Sign in with Google
- linking social account ke user lokal
- login dengan lebih dari satu metode
- tetap memakai session lokal yang sama

---

## Out of Scope

Untuk tahap ini, hal-hal berikut belum menjadi prioritas:

- Authentik
- self-hosted identity provider
- single sign-on lintas banyak aplikasi
- MFA
- forgot password
- email verification
- enterprise SAML / OIDC provider mode
- billing authorization
- fine-grained permission per action yang sangat detail
- delegated organization management yang kompleks

---

## 5. Dependency Strategy

## Principle

Sistem auth harus dibangun dengan dependency seminimal mungkin.

Tujuannya:

- mengurangi abstraction yang tidak perlu
- memudahkan debugging
- membuat fondasi auth lebih mudah dipahami
- menjaga kontrol penuh atas flow session dan authorization

## Dependency policy

### Yang dipakai

- standard library Go, terutama untuk HTTP, cookies, dan request handling
- `golang.org/x/crypto/argon2` untuk password hashing
- `golang.org/x/oauth2` untuk social login provider di fase 2

### Yang tidak dipakai di fase awal

- `gorilla/sessions`
- `scs`
- `alexedwards/argon2id`
- Authentik
- library auth all-in-one lain yang menyembunyikan terlalu banyak flow internal

## Rationale

Dependency tambahan seperti `scs`, `gorilla/sessions`, atau wrapper Argon2 memang bisa mempercepat implementasi, tetapi untuk fase awal produk ini, kebutuhan auth masih cukup sederhana dan lebih baik dibangun dengan fondasi yang eksplisit dan mudah dipahami.

---

## 6. User Model

Sistem harus membedakan dua level user role:

### 6.1 Platform-level role

Ini adalah role internal milik operator produk.

Contoh:

- user
- super_admin
- support_admin

Fungsi role ini:

- akses operasional internal
- moderation
- support
- audit
- investigasi abuse

Role ini **bukan** role utama untuk penggunaan normal PaaS.

---

### 6.2 Workspace-level role

Ini adalah role yang dipakai untuk penggunaan produk sehari-hari.

Contoh:

- owner
- admin
- developer
- viewer

Makna awal role:

- **owner**: kontrol penuh atas workspace
- **admin**: bisa mengelola sebagian besar konfigurasi workspace
- **developer**: bisa mengelola app dan deploy
- **viewer**: hanya bisa melihat resource tertentu

Pada fase awal, role yang wajib ada hanya:

- owner
- developer
- viewer

Role `admin` bisa ditambahkan jika memang diperlukan.

---

## 7. Tenant Model

Produk harus dibangun sebagai **multi-tenant system**.

### Tenant boundary

Tenant utama adalah **workspace**.

Setiap resource penting harus terikat ke satu workspace, misalnya:

- projects
- servers
- deployments
- domains
- environment variables
- logs metadata
- team members

Dengan model ini, sistem dapat selalu menjawab pertanyaan:

- user ini masuk workspace mana?
- user ini role-nya apa di workspace tersebut?
- user ini boleh melakukan aksi ini atau tidak?

---

## 8. Core Data Model

## 8.1 Users

Menyimpan akun global user.

Field konseptual:

- user id
- email
- password hash
- platform role
- status aktif/nonaktif
- created at
- updated at

## 8.2 Workspaces

Menyimpan tenant/account utama.

Field konseptual:

- workspace id
- workspace name
- owner user id
- created at
- updated at

## 8.3 Workspace Memberships

Relasi user dengan workspace.

Field konseptual:

- membership id
- workspace id
- user id
- workspace role
- created at

## 8.4 OAuth Identities

Dipakai di phase 2 untuk menghubungkan akun lokal dengan provider social login.

Field konseptual:

- identity id
- user id
- provider
- provider user id
- provider email
- created at

---

## 9. Resource Ownership Rule

Semua resource yang bisa dioperasikan user harus punya referensi ke workspace.

Contohnya:

- project harus punya workspace
- server harus punya workspace
- deployment harus punya workspace
- domain harus punya workspace

Dengan begitu authorization tidak bergantung pada asumsi global, tetapi pada ownership resource.

---

## 10. Phase 1 Requirements

## 10.1 Registration

User dapat membuat akun dengan email dan password.

Setelah registrasi berhasil, sistem akan:

1. membuat user baru
2. membuat workspace pertama secara otomatis
3. menambahkan user sebagai owner workspace tersebut
4. membuat session login

### Notes

Untuk fase awal, registrasi bisa tetap dibatasi dengan salah satu opsi berikut:

- open registration
- invite-only
- waitlist approval

Keputusan ini bisa diatur lewat config produk, bukan lewat perubahan arsitektur.

---

## 10.2 Login

User dapat login menggunakan:

- email
- password

Setelah login berhasil, sistem membuat session dan menyimpan context dasar user.

Session minimal perlu mengetahui:

- user id
- active workspace id
- role user di workspace aktif

---

## 10.3 Logout

User dapat logout dan session harus dihapus dengan benar.

---

## 10.4 Default Workspace Behavior

Setelah register, user otomatis memiliki satu workspace pertama.

Tujuannya:

- mengurangi friction onboarding
- user bisa langsung mulai membuat resource
- tidak perlu wizard kompleks di awal

Nama workspace awal bisa mengikuti:

- nama email user
- nama personal default
- atau naming sederhana yang bisa diubah nanti

---

## 10.5 Authorization in Phase 1

Pada phase 1, authorization cukup memakai rule dasar berikut:

- hanya member workspace yang bisa mengakses resource workspace
- hanya owner/developer yang bisa melakukan deploy
- hanya owner yang bisa mengelola membership
- viewer hanya bisa melihat resource yang memang diizinkan

Fokus phase 1 bukan membuat permission matrix yang rumit, tetapi memastikan bahwa boundary workspace sudah benar.

---

## 11. Phase 2 Requirements

## 11.1 Social Login

Tambahkan dua metode login:

- Continue with GitHub
- Continue with Google

Tujuannya:

- menurunkan friction login
- memudahkan onboarding
- memberi jalur login tambahan tanpa mengubah fondasi session

---

## 11.2 Account Linking

Sistem harus mendukung hubungan antara user lokal dan provider social login.

Aturan awal:

- jika provider identity sudah pernah dipakai, login ke user yang sama
- jika email provider cocok dengan akun lokal yang sudah ada, identity dihubungkan ke akun lokal itu
- jika belum ada akun sama sekali, sistem membuat user baru lalu membuat workspace awal

Dengan strategi ini:

- user lama email/password bisa menambah GitHub/Google
- user baru bisa langsung masuk lewat social login
- sistem tetap punya satu identitas user lokal yang konsisten

---

## 11.3 Session Consistency

Walaupun metode login bertambah, session lokal tetap satu sistem yang sama.

Artinya:

- email/password login dan social login tetap berujung ke session lokal milik app
- authorization tidak berubah
- workspace model tidak berubah

---

## 12. Security Requirements

## Phase 1 minimum security

- password harus di-hash dengan Argon2id
- password tidak boleh disimpan dalam plaintext
- session harus disimpan dalam cookie yang aman
- cookie harus memakai HttpOnly
- Secure harus aktif di production
- SameSite harus disetel dengan aman
- session harus punya expiration
- session harus di-rotate setelah login
- logout harus benar-benar menghapus session

## Phase 2 additional security

- OAuth state wajib divalidasi
- callback hanya menerima request yang valid
- identity provider response harus diverifikasi
- account linking tidak boleh membuat account takeover

---

## 13. Non-Functional Requirements

- auth flow harus sederhana dipahami oleh tim
- dependency harus sesedikit mungkin
- flow harus mudah di-debug
- session logic harus bisa di-upgrade nanti ke Redis atau DB-backed session jika diperlukan
- model auth harus kompatibel dengan pertumbuhan ke multi-user SaaS
- social login tidak boleh memaksa migrasi besar pada authorization model

---

## 14. UX Principles

- onboarding harus serendah mungkin friksinya
- user tidak boleh merasa sedang memakai “panel admin”
- register harus terasa seperti mendaftar ke SaaS biasa
- setelah login pertama, user harus langsung punya workspace dan bisa mulai menggunakan produk
- social login harus menjadi opsi tambahan, bukan satu-satunya jalan

---

## 15. Rollout Plan

## Phase 1

Target:

- auth dasar untuk user umum
- workspace pertama otomatis
- session lokal
- resource ownership mulai konsisten

Hasil:

- produk sudah siap dipakai user personal pertama

## Phase 2

Target:

- tambah GitHub login
- tambah Google login
- provider identity mapping

Hasil:

- friction login turun
- onboarding lebih mudah
- tetap tanpa perlu membawa Authentik/OIDC provider penuh

## Future Phase

Baru dipertimbangkan setelah produk valid:

- MFA
- forgot password
- email verification
- invite flow yang proper
- team management yang lebih matang
- organization/workspace switching yang lebih kaya
- Authentik atau external identity provider
- enterprise SSO

---

## 16. Risks

### Risk 1 — terlalu cepat membuat auth kompleks

Kalau langsung masuk ke Authentik/OIDC penuh, tim bisa terdistraksi dari problem inti PaaS.

### Risk 2 — auth masih dipikirkan sebagai admin panel

Kalau data model masih admin-centric, nanti migrasi ke multi-user akan jauh lebih mahal.

### Risk 3 — account linking salah desain

Linking GitHub/Google ke akun lokal harus hati-hati agar tidak membuka celah takeover.

### Risk 4 — permission model terlalu rumit terlalu cepat

Kalau role/permission dibuat terlalu detail di awal, implementasi akan lambat dan sulit divalidasi.

---

## 17. Success Criteria

Phase 1 dianggap berhasil jika:

- user bisa register
- user bisa login
- user otomatis punya workspace pertama
- user hanya bisa mengakses resource workspace miliknya
- deploy route sudah memakai authorization berbasis workspace

Phase 2 dianggap berhasil jika:

- user bisa login via GitHub
- user bisa login via Google
- identity provider bisa dipetakan ke akun lokal
- session dan authorization tetap konsisten
- onboarding terasa lebih ringan dibanding phase 1

---

## 18. Final Recommendation

Fondasi auth untuk produk ini harus diposisikan sebagai **multi-user SaaS auth**, bukan admin-only auth.

Keputusan implementasi yang direkomendasikan:

- mulai dari email/password + session lokal
- bangun model `users + workspaces + memberships` dari awal
- pakai dependency seminimal mungkin
- tambah GitHub/Google di fase berikutnya
- tunda Authentik/OIDC provider penuh sampai produk benar-benar membutuhkannya

Dengan pendekatan ini, sistem tetap sederhana untuk dibangun sekarang, tetapi arah arsitekturnya sudah benar untuk pertumbuhan produk PaaS di masa depan.
