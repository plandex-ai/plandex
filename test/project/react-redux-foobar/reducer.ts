import { createReducer } from '@reduxjs/toolkit';
import { INCREMENT_COUNT } from './action';

interface State {
  count: number;
}

const initialState: State = {
  count: 0,
};

const countReducer = createReducer(initialState, (builder) => {
  builder.addCase(INCREMENT_COUNT, (state, action) => {
    state.count += action.payload;
  });
});

export default countReducer;
