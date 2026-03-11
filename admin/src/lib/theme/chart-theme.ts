export const CHART_COLORS = {
  primary: 'hsl(160, 84%, 39%)',
  secondary: 'hsl(190, 80%, 45%)',
  tertiary: 'hsl(270, 60%, 55%)',
  warning: 'hsl(35, 90%, 55%)',
  danger: 'hsl(0, 75%, 55%)',
  muted: 'hsl(0, 0%, 60%)',
} as const;

export const CHART_COLORS_ARRAY = [
  CHART_COLORS.primary,
  CHART_COLORS.secondary,
  CHART_COLORS.tertiary,
  CHART_COLORS.warning,
  CHART_COLORS.danger,
  CHART_COLORS.muted,
] as const;
