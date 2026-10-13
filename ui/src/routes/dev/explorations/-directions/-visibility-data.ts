import { type Filters, metrics, type Observation, selectObservations } from "../-lib/data";

export function visibilitySeries(
  observations: Observation[],
  filters: Filters,
  brands: string[],
  promptId: string | null,
) {
  const answers = selectObservations(observations, filters).filter(
    (answer) => promptId === null || answer.promptId === promptId,
  );
  const days = Array.from({ length: filters.days }, (_, index) => 180 - filters.days + index);
  const daily = days.map((day) => answers.filter((answer) => answer.day === day));

  return brands.map((brand) => ({
    brand,
    points: daily.map((answers, index) => {
      const summary = metrics(answers, brand);
      return { day: days[index], value: summary.completed ? summary.presence : null };
    }),
  }));
}
