// dayKindNote explains an off day, a holiday or a working day in one sentence, for a day
// that carries no item of its own.
export const dayKindNote = (kind) => {
  if (!kind) return ''
  if (kind.working_day) {
    return `The office made this a working day, so it follows the ${kind.follows_weekday_name} routine.`
  }
  return 'No fixed routine is generated for this day. A class or exam created for it is still kept and announced.'
}
