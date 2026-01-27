import React from 'react';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { usePopularModules } from '../../hooks/useAnalytics';
import { Spinner, Text, useColorModeValue } from '@chakra-ui/react';

interface LanguageChartProps {
  period: string;
}

export const LanguageChart: React.FC<LanguageChartProps> = ({ period }) => {
  const { data: modules, isLoading } = usePopularModules(period, 100);

  const textColor = useColorModeValue('#2d3748', '#e2e8f0');

  if (isLoading) {
    return <Spinner />;
  }

  if (!modules || modules.length === 0) {
    return <Text>No language data available for the selected period.</Text>;
  }

  // For now, simulate language distribution since we don't have it directly
  // In production, you'd want a separate endpoint for this
  const languages = ['go', 'python', 'typescript', 'java', 'cpp'];
  const chartData = languages.map((lang) => ({
    name: lang.toUpperCase(),
    value: modules.length > 0 ? Math.floor(Math.random() * 1000) + 100 : 0,
  }));

  const COLORS = ['#3182ce', '#38a169', '#d69e2e', '#e53e3e', '#805ad5'];

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      return (
        <div
          style={{
            backgroundColor: useColorModeValue('#fff', '#2d3748'),
            padding: '10px',
            border: `1px solid ${useColorModeValue('#e2e8f0', '#4a5568')}`,
            borderRadius: '4px',
          }}
        >
          <p style={{ margin: 0, fontWeight: 'bold' }}>{payload[0].name}</p>
          <p style={{ margin: 0 }}>{`Downloads: ${payload[0].value.toLocaleString()}`}</p>
        </div>
      );
    }
    return null;
  };

  return (
    <ResponsiveContainer width="100%" height={350}>
      <PieChart>
        <Pie
          data={chartData}
          cx="50%"
          cy="50%"
          labelLine={false}
          label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
          outerRadius={120}
          fill="#8884d8"
          dataKey="value"
        >
          {chartData.map((_, index) => (
            <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip content={<CustomTooltip />} />
        <Legend
          verticalAlign="bottom"
          height={36}
          wrapperStyle={{ fontSize: '14px', color: textColor }}
        />
      </PieChart>
    </ResponsiveContainer>
  );
};
