interface Props {
  status: string;
}

const statusColors: Record<string, string> = {
  COMPLETED: 'bg-emerald-900/50 text-emerald-400 border-emerald-700',
  RETRY_SUCCESS: 'bg-yellow-900/50 text-yellow-400 border-yellow-700',
  FAILED: 'bg-red-900/50 text-red-400 border-red-700',
  PROCESSING: 'bg-blue-900/50 text-blue-400 border-blue-700',
  RECEIVED: 'bg-gray-800 text-gray-400 border-gray-700',
  SUCCESS: 'bg-emerald-900/50 text-emerald-400 border-emerald-700',
  RETRYING: 'bg-yellow-900/50 text-yellow-400 border-yellow-700',
};

export default function StatusBadge({ status }: Props) {
  const cls = statusColors[status] || 'bg-gray-800 text-gray-400 border-gray-700';
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium border ${cls}`}>
      {status}
    </span>
  );
}
