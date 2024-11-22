import React, { useState, useEffect, useCallback, useRef } from 'react';
import type { FC, ReactNode, FormEvent } from 'react';

// Type definitions
interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'user';
}

type ValidationResult = {
  valid: boolean;
  errors: string[];
};

// Props interface with generic type
interface DataListProps<T> {
  items: T[];
  renderItem: (item: T) => ReactNode;
  onItemSelect?: (item: T) => void;
}

// Custom hook with TypeScript
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);

  return debouncedValue;
}

// Higher-order component with TypeScript
function withLoading<P extends object>(
  WrappedComponent: React.ComponentType<P>
): FC<P & { loading?: boolean }> {
  return ({ loading = false, ...props }) => (
    <div className="wrapper">
      {loading ? (
        <div className="loading-spinner" />
      ) : (
        <WrappedComponent {...(props as P)} />
      )}
    </div>
  );
}

// Form component with controlled inputs
const UserForm: FC<{ onSubmit: (user: Partial<User>) => void }> = ({ onSubmit }) => {
  const [formData, setFormData] = useState<Partial<User>>({
    name: '',
    email: '',
    role: 'user'
  });

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  return (
    <form onSubmit={handleSubmit} className="user-form">
      <div className="form-group">
        <label htmlFor="name">Name:</label>
        <input
          type="text"
          id="name"
          value={formData.name}
          onChange={e => setFormData(prev => ({
            ...prev,
            name: e.target.value
          }))}
          required
        />
      </div>

      <div className="form-group">
        <label htmlFor="email">Email:</label>
        <input
          type="email"
          id="email"
          value={formData.email}
          onChange={e => setFormData(prev => ({
            ...prev,
            email: e.target.value
          }))}
          required
        />
      </div>

      <div className="form-group">
        <label htmlFor="role">Role:</label>
        <select
          id="role"
          value={formData.role}
          onChange={e => setFormData(prev => ({
            ...prev,
            role: e.target.value as User['role']
          }))}
        >
          <option value="user">User</option>
          <option value="admin">Admin</option>
        </select>
      </div>

      <button type="submit">Submit</button>
    </form>
  );
};

// Generic list component
const DataList = <T extends { id: string }>({
  items,
  renderItem,
  onItemSelect
}: DataListProps<T>): JSX.Element => {
  return (
    <ul className="data-list">
      {items.map(item => (
        <li
          key={item.id}
          onClick={() => onItemSelect?.(item)}
          role="button"
          tabIndex={0}
        >
          {renderItem(item)}
        </li>
      ))}
    </ul>
  );
};

// Main app component using all features
const App: FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const debouncedSearch = useDebounce(searchTerm, 300);
  const formRef = useRef<HTMLFormElement>(null);

  const handleUserSubmit = useCallback((userData: Partial<User>) => {
    const newUser: User = {
      id: crypto.randomUUID(),
      ...userData as Omit<User, 'id'>
    };
    setUsers(prev => [...prev, newUser]);
  }, []);

  const filteredUsers = users.filter(user =>
    user.name.toLowerCase().includes(debouncedSearch.toLowerCase())
  );

  return (
    <div className="app">
      <header className="app-header">
        <h1>User Management</h1>
        <input
          type="search"
          placeholder="Search users..."
          value={searchTerm}
          onChange={e => setSearchTerm(e.target.value)}
          className="search-input"
        />
      </header>

      <main>
        <section className="user-form-section">
          <h2>Add New User</h2>
          <UserForm onSubmit={handleUserSubmit} />
        </section>

        <section className="user-list-section">
          <h2>Users</h2>
          <DataList
            items={filteredUsers}
            renderItem={user => (
              <div className="user-item">
                <strong>{user.name}</strong>
                <span>{user.email}</span>
                <badge className={`role-badge ${user.role}`}>
                  {user.role}
                </badge>
              </div>
            )}
            onItemSelect={user => console.log('Selected user:', user)}
          />
        </section>
      </main>
    </div>
  );
};

export default App;
