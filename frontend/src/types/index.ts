export interface Config {
  id: number;
  trade_amount_usd: number;
  leverage: number;
  margin_mode: string;
  commission_rate: number;
  updated_at: string;
}

export interface Trade {
  id: number;
  coin: string;
  signal_type: string;
  side: string;
  account_type: string;
  quantity: number;
  entry_price: number;
  leverage: number;
  commission: number;
  is_active: boolean;
  binance_order_id: number | null;
  created_at: string;
}

export interface PnLRecord {
  id: number;
  coin: string;
  closed_trade_id: number;
  closing_trade_id: number;
  closed_signal_type: string;
  closing_signal_type: string;
  entry_price: number;
  exit_price: number;
  quantity: number;
  leverage: number;
  gross_pnl: number;
  open_commission: number;
  close_commission: number;
  total_commission: number;
  net_pnl: number;
  side: string;
  closed_at: string;
}

export interface PnLSeries {
  series_index: number;
  coin: string;
  side: string;
  sequence: string[];
  total_gross_pnl: number;
  total_commission: number;
  total_net_pnl: number;
  records: PnLRecord[];
}

export interface PnLSummary {
  coin: string;
  side: string;
  trade_count: number;
  total_gross_pnl: number;
  total_commission: number;
  total_net_pnl: number;
  win_count: number;
  loss_count: number;
  win_rate: number;
}

export interface UnrealizedPnLItem {
  trade_id: number;
  coin: string;
  signal_type: string;
  side: string;
  account_type: string;
  quantity: number;
  entry_price: number;
  current_price: number;
  leverage: number;
  gross_pnl: number;
  open_commission: number;
  close_commission: number;
  total_commission: number;
  net_pnl: number;
}

export interface UnrealizedPnLResponse {
  items: UnrealizedPnLItem[];
  total_gross_pnl: number;
  total_commission: number;
  total_net_pnl: number;
  updated_at: string;
}

export interface PnL24hAccountSummary {
  account_type: string;
  label: string;
  trade_count: number;
  realized_gross_pnl: number;
  realized_commission: number;
  realized_net_pnl: number;
  unrealized_net_pnl: number;
  total_net_pnl: number;
  win_count: number;
  loss_count: number;
  win_rate: number;
}

export interface PnL24hCoinSummary {
  coin: string;
  trade_count: number;
  realized_gross_pnl: number;
  realized_commission: number;
  realized_net_pnl: number;
  unrealized_net_pnl: number;
  total_net_pnl: number;
  al_net_pnl: number;
  sat_net_pnl: number;
}

export interface PnL24hChartPoint {
  time: string;
  realized_net_pnl: number;
  cumulative_realized_pnl: number;
}

export interface PnL24hResponse {
  from: string;
  to: string;
  updated_at: string;
  accounts: PnL24hAccountSummary[];
  coins: PnL24hCoinSummary[];
  chart: PnL24hChartPoint[];
  total_realized_gross_pnl: number;
  total_realized_commission: number;
  total_realized_net_pnl: number;
  total_unrealized_net_pnl: number;
  total_net_pnl: number;
}

export interface ExecutionLog {
  id: number;
  webhook_id: number;
  attempt_number: number;
  step: string;
  status: string;
  account_type: string | null;
  binance_request: unknown;
  binance_response: unknown;
  error_category: string | null;
  error_message: string | null;
  duration_ms: number | null;
  created_at: string;
}

export interface WebhookLog {
  id: number;
  request_id: string;
  coin: string;
  signal_type: string;
  raw_body: unknown;
  status: string;
  trade_id: number | null;
  error_message: string | null;
  retry_count: number;
  received_at: string;
  completed_at: string | null;
  executions?: ExecutionLog[];
}

export interface ErrorCategory {
  category: string;
  count: number;
}

export interface SystemStats {
  total_webhooks: number;
  completed_count: number;
  retry_success_count: number;
  failed_count: number;
  processing_count: number;
  success_rate: number;
  retry_rate: number;
  fail_rate: number;
  avg_retry_count: number;
  error_breakdown: ErrorCategory[];
  failed_details: WebhookLog[];
}
