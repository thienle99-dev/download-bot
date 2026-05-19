<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../lib/api';
  import { showToast } from '../lib/stores/toast';
  import type { AIConfig } from '../lib/types';

  let config: AIConfig = {
    base_url: '',
    api_key: '',
    model: '',
    system_prompt: '',
    enabled: false
  };

  let models: string[] = [];
  let loading = true;
  let saving = false;
  let loadingModels = false;

  async function loadConfig() {
    try {
      config = await api.getAIConfig();
      if (config.base_url && config.api_key) {
        // Automatically fetch models if URL and key are set
        await fetchModels(true);
      }
    } catch (err: any) {
      showToast('error', err.message || 'Lỗi khi tải cấu hình AI.');
    } finally {
      loading = false;
    }
  }

  async function fetchModels(silent = false) {
    if (!config.base_url || !config.api_key) {
      if (!silent) {
        showToast('error', 'Vui lòng nhập Base URL và API Key trước khi tải danh sách Model.');
      }
      return;
    }

    loadingModels = true;
    try {
      // Temporarily save current input config so backend can query using updated key/url if they changed
      if (!silent) {
        await api.saveAIConfig(config);
      }
      const res = await api.getAIModels();
      models = res.models;
      if (models.length > 0 && !config.model) {
        config.model = models[0];
      }
      if (!silent) {
        showToast('success', `Tải thành công ${models.length} models.`);
      }
    } catch (err: any) {
      if (!silent) {
        showToast('error', err.message || 'Lỗi khi tải danh sách Model. Đảm bảo API Key và Base URL chính xác.');
      }
    } finally {
      loadingModels = false;
    }
  }

  async function saveConfig() {
    saving = true;
    try {
      await api.saveAIConfig(config);
      showToast('success', 'Đã lưu cấu hình AI thành công.');
    } catch (err: any) {
      showToast('error', err.message || 'Không thể lưu cấu hình AI.');
    } finally {
      saving = false;
    }
  }

  onMount(loadConfig);
</script>

{#if loading}
  <div class="h-full flex items-center justify-center p-8">
    <div class="flex flex-col items-center gap-3">
      <span class="w-8 h-8 rounded-full border-4 border-sky-500/20 border-t-sky-500 animate-spin"></span>
      <p class="text-sm font-semibold text-slate-500 dark:text-slate-400">Đang tải cấu hình AI...</p>
    </div>
  </div>
{:else}
  <div class="p-6 max-w-4xl mx-auto space-y-6">
    <div class="flex flex-col gap-2">
      <h2 class="font-heading font-bold text-2xl text-slate-800 dark:text-slate-100 flex items-center gap-2">
        <span>🤖</span> Cấu hình Trợ lý AI
      </h2>
      <p class="text-xs text-slate-500 dark:text-slate-400">
        Cho phép người dùng sử dụng lệnh <code>/ai [câu hỏi]</code> trên Telegram để giao tiếp với các mô hình ngôn ngữ lớn tương thích OpenAI-compatible API.
      </p>
    </div>

    <div class="glass p-6 rounded-2xl border border-slate-200/10 space-y-6">
      <!-- Enabled Toggle Switch -->
      <div class="flex items-center justify-between p-4 rounded-xl bg-slate-900/40 border border-slate-800/50">
        <div>
          <h4 class="text-sm font-bold text-slate-100">Kích hoạt AI Chatbot</h4>
          <p class="text-xs text-slate-400">Bật/tắt tính năng chat với AI trên Telegram</p>
        </div>
        <!-- Toggle Button -->
        <!-- svelte-ignore a11y-role-supports-aria-props -->
        <button
          type="button"
          class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {config.enabled ? 'bg-sky-500' : 'bg-slate-700'}"
          role="switch"
          aria-label="Kích hoạt AI Chatbot"
          aria-checked={config.enabled}
          on:click={() => config.enabled = !config.enabled}
        >
          <span
            aria-hidden="true"
            class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {config.enabled ? 'translate-x-5' : 'translate-x-0'}"
          ></span>
        </button>
      </div>

      <!-- Config Form -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
        <!-- Base URL -->
        <div class="flex flex-col gap-2">
          <label for="base_url" class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Base URL</label>
          <input
            id="base_url"
            type="text"
            bind:value={config.base_url}
            placeholder="https://api.openai.com/v1"
            class="px-4 py-2.5 rounded-xl border border-slate-200/10 bg-slate-950/50 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition-colors text-sm"
          />
        </div>

        <!-- API Key -->
        <div class="flex flex-col gap-2">
          <label for="api_key" class="text-xs font-semibold text-slate-400 uppercase tracking-wider">API Key</label>
          <input
            id="api_key"
            type="password"
            bind:value={config.api_key}
            placeholder="Nhập API Key hoặc token..."
            class="px-4 py-2.5 rounded-xl border border-slate-200/10 bg-slate-950/50 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition-colors text-sm"
          />
        </div>

        <!-- Model Selection -->
        <div class="flex flex-col gap-2 md:col-span-2">
          <label for="model" class="text-xs font-semibold text-slate-400 uppercase tracking-wider">Model</label>
          <div class="flex gap-2">
            {#if models.length > 0}
              <select
                id="model"
                bind:value={config.model}
                class="flex-grow px-4 py-2.5 rounded-xl border border-slate-200/10 bg-slate-950/50 text-slate-100 focus:outline-none focus:border-sky-500 transition-colors text-sm"
              >
                {#each models as m}
                  <option value={m}>{m}</option>
                {/each}
              </select>
            {:else}
              <input
                id="model"
                type="text"
                bind:value={config.model}
                placeholder="Ví dụ: gpt-4o, claude-3-5-sonnet..."
                class="flex-grow px-4 py-2.5 rounded-xl border border-slate-200/10 bg-slate-950/50 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition-colors text-sm"
              />
            {/if}
            <button
              type="button"
              on:click={() => fetchModels(false)}
              disabled={loadingModels}
              class="px-4 rounded-xl text-xs font-bold bg-slate-800 hover:bg-slate-700 text-sky-400 transition-colors border border-slate-700 whitespace-nowrap"
            >
              {#if loadingModels}
                <span class="inline-block w-3 h-3 rounded-full border-2 border-sky-400/20 border-t-sky-400 animate-spin mr-1"></span>
              {/if}
              Tải Models
            </button>
          </div>
        </div>

        <!-- System Prompt -->
        <div class="flex flex-col gap-2 md:col-span-2">
          <label for="system_prompt" class="text-xs font-semibold text-slate-400 uppercase tracking-wider">System Prompt (Nhân cách AI)</label>
          <textarea
            id="system_prompt"
            rows="4"
            bind:value={config.system_prompt}
            placeholder="Bạn là trợ lý AI hữu ích..."
            class="px-4 py-2.5 rounded-xl border border-slate-200/10 bg-slate-950/50 text-slate-100 placeholder-slate-600 focus:outline-none focus:border-sky-500 transition-colors text-sm resize-none"
          ></textarea>
        </div>
      </div>

      <!-- Action Button -->
      <div class="flex justify-end pt-4 border-t border-slate-200/10">
        <button
          type="button"
          on:click={saveConfig}
          disabled={saving}
          class="flex items-center gap-2 px-6 py-2.5 rounded-xl text-sm font-bold text-white bg-sky-500 hover:bg-sky-600 disabled:bg-sky-500/50 shadow-md shadow-sky-500/10 hover:shadow-sky-500/20 transition-all duration-200"
        >
          {#if saving}
            <span class="w-4 h-4 rounded-full border-2 border-white/20 border-t-white animate-spin"></span>
            Đang lưu...
          {:else}
            💾 Lưu cấu hình
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
