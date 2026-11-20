import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';

export interface SeriesSpec {
  label: string;
  stroke: string;
  dash?: number[];
}

export interface Chart {
  update(xs: number[], ys: (number | null)[][]): void;
  destroy(): void;
}

export function lineChart(el: HTMLElement, title: string, series: SeriesSpec[], yLabel: string): Chart {
  const empty = [[], ...series.map(() => [])] as unknown as uPlot.AlignedData;
  const plot = new uPlot(
    {
      title,
      width: el.clientWidth || 600,
      height: 240,
      scales: { x: { time: false } },
      series: [
        { label: 'sec' },
        ...series.map((s) => ({ label: s.label, stroke: s.stroke, width: 2, dash: s.dash, spanGaps: true })),
      ] as uPlot.Series[],
      axes: [{ label: 'seconds' }, { label: yLabel }],
    },
    empty,
    el,
  );
  const onResize = () => plot.setSize({ width: el.clientWidth || 600, height: 240 });
  window.addEventListener('resize', onResize);
  return {
    update: (xs, ys) => plot.setData([xs, ...ys] as unknown as uPlot.AlignedData),
    destroy: () => {
      window.removeEventListener('resize', onResize);
      plot.destroy();
    },
  };
}
