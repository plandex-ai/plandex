<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { writable, derived } from 'svelte/store';
  import type { Writable } from 'svelte/store';

  // Props with TypeScript types
  export let title: string;
  export let initialCount: number = 0;

  // Reactive declarations
  $: doubled = count * 2;
  $: {
    if (count > 10) {
      console.log('Count is getting high!');
    }
  }

  // Local state
  let count: number = initialCount;
  let inputValue: string = '';
  let mounted: boolean = false;

  // Stores
  const items: Writable<string[]> = writable([]);
  const filteredItems = derived(items, $items => 
    $items.filter(item => item.includes(inputValue))
  );

  // Event handlers
  function handleClick() {
    count += 1;
  }

  function addItem() {
    if (inputValue.trim()) {
      items.update(items => [...items, inputValue.trim()]);
      inputValue = '';
    }
  }

  function removeItem(index: number) {
    items.update(items => items.filter((_, i) => i !== index));
  }

  // Lifecycle
  onMount(() => {
    mounted = true;
    return () => {
      mounted = false;
    };
  });

  onDestroy(() => {
    console.log('Component destroyed');
  });
</script>

<!-- Markup section -->
<div class="container">
  <h1>{title}</h1>

  <!-- Event binding and reactive values -->
  <div class="counter">
    <button on:click={handleClick}>
      Count: {count}
    </button>
    <p>Doubled: {doubled}</p>
  </div>

  <!-- Form with two-way binding -->
  <div class="form">
    <input
      type="text"
      bind:value={inputValue}
      placeholder="Add item"
      on:keydown={e => e.key === 'Enter' && addItem()}
    />
    <button on:click={addItem}>Add</button>
  </div>

  <!-- Conditional rendering -->
  {#if $items.length > 0}
    <!-- List with store subscription -->
    <ul>
      {#each $filteredItems as item, index (item)}
        <li class="item">
          {item}
          <button on:click={() => removeItem(index)}>×</button>
        </li>
      {/each}
    </ul>
  {:else}
    <p>No items added yet</p>
  {/if}

  <!-- Slots for content projection -->
  <div class="content">
    <slot name="header">
      <h2>Default Header</h2>
    </slot>
    
    <slot>
      <p>Default content</p>
    </slot>
    
    <slot name="footer" />
  </div>
</div>

<style>
  .container {
    padding: 1rem;
    max-width: 600px;
    margin: 0 auto;
  }

  .counter {
    margin: 1rem 0;
  }

  .form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 4px;
  }

  button {
    padding: 0.5rem 1rem;
    background: #4CAF50;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  button:hover {
    background: #45a049;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  .item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem;
    margin: 0.25rem 0;
    background: #f5f5f5;
    border-radius: 4px;
  }

  .item button {
    background: #ff4444;
    padding: 0.25rem 0.5rem;
  }

  .item button:hover {
    background: #cc0000;
  }

  .content {
    margin-top: 2rem;
    padding: 1rem;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  /* Scoped styles - only apply to this component */
  :global(.theme-dark) .container {
    background: #333;
    color: #fff;
  }
</style>
