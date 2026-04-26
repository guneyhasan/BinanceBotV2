interface CardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  color?: 'green' | 'red' | 'blue' | 'yellow' | 'gray';
}

const colorMap = {
  green: 'text-emerald-400',
  red: 'text-red-400',
  blue: 'text-blue-400',
  yellow: 'text-yellow-400',
  gray: 'text-gray-400',
};

export default function Card({ title, value, subtitle, color = 'gray' }: CardProps) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
      <p className="text-sm text-gray-500 mb-1">{title}</p>
      <p className={`text-2xl font-bold ${colorMap[color]}`}>{value}</p>
      {subtitle && <p className="text-xs text-gray-600 mt-1">{subtitle}</p>}
    </div>
  );
}
