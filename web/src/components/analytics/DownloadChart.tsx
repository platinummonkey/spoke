import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { usePopularModules } from '../../hooks/useAnalytics';
import { Spinner, Text, useColorModeValue } from '@chakra-ui/react';

interface DownloadChartProps {
  period: string;
}

export const DownloadChart: React.FC<DownloadChartProps> = ({ period }) => {
  const { data: modules, isLoading } = usePopularModules(period, 5);

  const lineColor = useColorModeValue('#3182ce', '#63b3ed');
  const gridColor = useColorModeValue('#e2e8f0', '#4a5568');
  const textColor = useColorModeValue('#2d3748', '#e2e8f0');

  if (isLoading) {
    return <Spinner />;
  }

  if (!modules || modules.length === 0) {
    return <Text>No download data available for the selected period.</Text>;
  }

  // Transform data for chart
  const chartData = modules.map((module) => ({
    name: module.module_name,
    downloads: module.total_downloads,
    avgDaily: Math.round(module.avg_daily_downloads),
  }));

  return (
    <ResponsiveContainer width="100%" height={350}>
      <LineChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
        <XAxis
          dataKey="name"
          angle={-45}
          textAnchor="end"
          height={100}
          tick={{ fill: textColor, fontSize: 12 }}
        />
        <YAxis tick={{ fill: textColor }} />
        <Tooltip
          contentStyle={{
            backgroundColor: useColorModeValue('#fff', '#2d3748'),
            border: `1px solid ${gridColor}`,
            borderRadius: '4px',
          }}
        />
        <Legend />
        <Line
          type="monotone"
          dataKey="downloads"
          stroke={lineColor}
          strokeWidth={2}
          dot={{ r: 4 }}
          name="Total Downloads"
        />
        <Line
          type="monotone"
          dataKey="avgDaily"
          stroke="#38a169"
          strokeWidth={2}
          dot={{ r: 4 }}
          name="Avg Daily"
        />
      </LineChart>
    </ResponsiveContainer>
  );
};
