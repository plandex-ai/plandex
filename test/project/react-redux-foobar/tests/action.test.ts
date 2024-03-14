import { myAction, MY_ACTION_TYPE } from '../action';

describe('myAction', () => {
  it('creates an action with the correct type and payload', () => {
    const payload = {}; // Define payload
    const expectedAction = {
      type: MY_ACTION_TYPE,
      payload,
    };
    expect(myAction(payload)).toEqual(expectedAction);
  });
});
