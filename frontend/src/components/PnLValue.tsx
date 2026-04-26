interface Props {
  value: number;
  suffix?: string;
}

export default function PnLValue({ value, suffix = ' USD' }: Props) {
  const color = value > 0 ? 'text-emerald-400' : value < 0 ? 'text-red-400' : 'text-gray-400';
  const prefix = value > 0 ? '+' : '';
  return (
    <span className={`font-mono font-medium ${color}`}>
      {prefix}{value.toFixed(4)}{suffix}
    </span>
  );
}
