import { render, fireEvent } from '@testing-library/react';  it('increments count on button click', () => {
    const { getByText } = render(<MyComponent name="John Doe" />);
    fireEvent.click(getByText('Click me'));
    expect(getByText('You clicked 1 times')).toBeInTheDocument();
  });
});
import MyComponent from '../component';

describe('MyComponent', () => {
  it('renders with the correct name', () => {
    const { getByText } = render(<MyComponent name="John Doe" />);
    expect(getByText('Hello, John Doe!')).toBeInTheDocument();
  });
});

