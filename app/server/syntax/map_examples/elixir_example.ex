defmodule ExampleApp do
  # Module attributes (constants)
  @default_timeout 5000
  @version "1.0.0"

  # Type specifications
  @type user :: %{
    id: integer,
    name: String.t(),
    email: String.t(),
    roles: [atom]
  }

  # Struct definition
  defstruct name: "", age: 0, email: nil

  # Behaviour definition
  @callback process(term) :: {:ok, term} | {:error, term}

  # Protocol implementation
  defimpl String.Chars, for: ExampleApp do
    def to_string(%{name: name}), do: "ExampleApp: #{name}"
  end

  # Public function with guards
  def calculate_age(birth_year) when is_integer(birth_year) do
    current_year = DateTime.utc_now().year
    current_year - birth_year
  end

  # Private function
  defp validate_email(email) do
    String.match?(email, ~r/^[^\s]+@[^\s]+\.[^\s]+$/)
  end

  # Function with pattern matching
  def handle_result({:ok, value}), do: "Success: #{value}"
  def handle_result({:error, reason}), do: "Error: #{reason}"
  def handle_result(_), do: "Unknown result"

  # Macro definition
  defmacro debug(expression) do
    quote do
      IO.puts "Debug: #{inspect(unquote(expression))}"
    end
  end

  # Using with for complex operations
  def create_user(params) do
    with {:ok, name} <- Map.fetch(params, "name"),
         {:ok, email} <- Map.fetch(params, "email"),
         true <- validate_email(email) do
      %ExampleApp{name: name, email: email}
    else
      :error -> {:error, "Missing required fields"}
      false -> {:error, "Invalid email format"}
    end
  end

  # GenServer implementation
  use GenServer

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @impl true
  def init(opts) do
    {:ok, opts}
  end

  @impl true
  def handle_call({:get_info}, _from, state) do
    {:reply, state, state}
  end

  @impl true
  def handle_cast({:update_info, new_info}, _state) do
    {:noreply, new_info}
  end

  # Using comprehensions
  def process_list(items) do
    for item <- items,
        is_binary(item),
        String.length(item) > 0,
        do: String.upcase(item)
  end

  # Exception definition
  defmodule CustomError do
    defexception message: "A custom error occurred"
  end

  # Function raising custom exception
  def raise_custom_error do
    raise CustomError, message: "Something went wrong"
  end
end
