export interface DownloadHistory {
  id: number;
  user_id: number;
  chat_id: number;
  url: string;
  platform: string;
  title: string;
  format: string;
  file_size: number;
  file_id: string;
  created_at: string;
  file_exist: boolean;
  download_url: string;
}

export interface UserStat {
  user_id: number;
  chat_id: number;
  download_count: number;
  last_download: string;
}

export interface Stats {
  total_downloads: number;
  total_users: number;
  storage_used: number;
  max_concurrent: number;
}

export interface SystemConfig {
  download_dir: string;
  cache_dir: string;
  db_path: string;
  max_concurrent: number;
  public_url: string;
  server_port: string;
}

export interface QueueItem {
  id: string;
  user_id: number;
  title: string;
  url: string;
  progress: number;
  started_at: string;
}

export interface LogMessage {
  timestamp: string;
  level: 'INFO' | 'WARN' | 'ERROR';
  message: string;
}

export interface AIConfig {
  base_url: string;
  api_key: string;
  model: string;
  system_prompt: string;
  enabled: boolean;
}

