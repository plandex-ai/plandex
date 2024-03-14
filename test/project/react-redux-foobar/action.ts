export const INCREMENT_COUNT = 'INCREMENT_COUNT';

interface IncrementCountAction {
  type: typeof INCREMENT_COUNT;
  payload: number; // Payload now specifically expects a number
}

export const incrementCount = (amount: number): IncrementCountAction => ({
  type: INCREMENT_COUNT,
  payload: amount,
});
