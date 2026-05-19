<script lang="ts">
  import { push } from "svelte-spa-router";
  import { login } from "../lib/stores/auth";
  import { showToast } from "../lib/stores/toast";

  let password = "";
  let loading = false;

  async function handleLogin() {
    if (!password) {
      showToast("error", "Vui lòng nhập mật khẩu admin!");
      return;
    }

    loading = true;
    try {
      // Direct local storage check first, but it will validate on first API call.
      // To provide standard feedback, let's login
      const success = login(password);
      if (success) {
        showToast("success", "Đăng nhập thành công!");
        push("/overview");
      } else {
        showToast("error", "Đăng nhập thất bại.");
      }
    } catch (err: any) {
      showToast("error", err.message || "Mật khẩu không hợp lệ.");
    } finally {
      loading = false;
    }
  }
</script>

<div
  class="min-h-screen flex items-center justify-center bg-slate-950 px-4 relative overflow-hidden"
>
  <!-- Dynamic blurred background shapes -->
  <div
    class="absolute w-96 h-96 rounded-full bg-sky-500/10 blur-3xl -top-12 -left-12"
  ></div>
  <div
    class="absolute w-96 h-96 rounded-full bg-indigo-500/10 blur-3xl -bottom-12 -right-12"
  ></div>

  <div class="w-full max-w-md relative z-10">
    <div class="glass p-8 rounded-3xl shadow-2xl flex flex-col items-center">
      <!-- Icon logo -->
      <div
        class="w-16 h-16 rounded-2xl bg-gradient-to-tr from-sky-500 to-indigo-500 flex items-center justify-center text-white font-bold text-3xl font-heading shadow-xl shadow-sky-500/25 mb-6"
      >
        ↓
      </div>

      <h2 class="font-heading font-extrabold text-2xl text-slate-100 mb-1">
        Download Bot Admin
      </h2>
      <p
        class="text-xs text-slate-400 font-medium uppercase tracking-wider mb-8"
      >
        Hệ Thống Quản Trị Trung Tâm
      </p>

      <form on:submit|preventDefault={handleLogin} class="w-full space-y-4">
        <div>
          <label
            for="password"
            class="block text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2"
            >Mật khẩu Admin</label
          >
          <input
            id="password"
            type="password"
            bind:value={password}
            placeholder="Nhập mật khẩu cấu hình..."
            class="w-full px-4 py-3.5 rounded-xl bg-slate-900 border border-slate-800 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 focus:ring-1 focus:ring-sky-500 transition-all font-medium"
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          class="w-full py-3.5 rounded-xl bg-gradient-to-r from-sky-500 to-indigo-500 hover:from-sky-400 hover:to-indigo-400 text-white font-semibold shadow-lg shadow-sky-500/20 active:scale-[0.98] transition-all disabled:opacity-50 disabled:pointer-events-none"
        >
          {loading ? "Đang xác thực..." : "Đăng Nhập"}
        </button>
      </form>
    </div>
  </div>
</div>
