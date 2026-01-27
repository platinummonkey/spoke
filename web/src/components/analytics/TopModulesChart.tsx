import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from 'recharts';
import { useColorModeValue } from '@chakra-ui/react';
import { PopularModule } from '../../hooks/useAnalytics';

interface TopModulesChartProps {
  modules: PopularModule[];
}

export const TopModulesChart: React.FC<TopModulesChartProps> = ({ modules }) => {
  const gridColor = useColorModeValue('#e2e8f0', '#4a5568');
  const textColor = useColorModeValue('#2d3748', '#e2e8f0');

  const chartData = modules.map((module) => ({
    name: module.module_name.length > 20
      ? module.module_name.substring(0, 17) + '...'
      : module.module_name,
    fullName: module.module_name,
    downloads: module.total_downloads,
    views: module.total_views,
  }));

  const colors = [
    '#3182ce',
    '#38a169',
    '#d69e2e',
    '#e53e3e',
    '#805ad5',
    '#dd6b20',
    '#319795',
    '#d53f8c',
    '#00b5d8',
    '#9f7aea',
  ];

  return (
    <ResponsiveContainer width="100%" height={350}>
      <BarChart data={chartData} layout="vertical">
        <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
        <XAxis type="number" tick={{ fill: textColor }} />
        <YAxis
          type="category"
          dataKey="name"
          width={150}
          tick={{ fill: textColor, fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: useColorModeValue('#fff', '#2d3748'),
            border: `1px solid ${gridColor}`,
            borderRadius: '4px',
          }}
          formatter={(value: number, name: string) => {
            if (name === 'downloads' || name === 'views') {
              return [value.toLocaleString(), name === 'downloads' ? 'Downloads' : 'Views'];
            }
            return [value, name];
          }}
          labelFormatter={(label) => {
            const item = chartData.find(d => d.name === label);
            return item?.fullName || label;
          }}
        />
        <Legend />
        <Bar dataKey="downloads" name="Downloads">
          {chartData.map((_, index) => (
            <Cell key={`cell-${index}`} fill={colors[index % colors.length]} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
};
