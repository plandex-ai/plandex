defmodule ExampleApp do                                                   
  # Module attributes                                                     
  @default_timeout 5000                                                   
  @version "1.0.0"                                                        
                                                                          
  # Protocol definition                                                   
  defprotocol Formatter do                                                
    @doc "Format the data for display"                                    
    def format(data)                                                      
  end                                                                     
                                                                          
  # Protocol implementation                                               
  defimpl Formatter, for: Map do                                          
    def format(data) do                                                   
      inspect(data, pretty: true)                                         
    end                                                                   
  end                                                                     
                                                                          
  # Struct definition                                                     
  defstruct name: "", age: 0, email: nil                                  
                                                                          
  # Exception definition                                                  
  defexception message: "A custom error occurred"                         
                                                                          
  # Callback definition                                                   
  @callback process(term) :: {:ok, term} | {:error, term}                 
  @macrocallback validate(term) :: Macro.t()                              
                                                                          
  # Public function                                                       
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
                                                                          
  # Private macro                                                         
  defmacrop log(message) do                                               
    quote do                                                              
      IO.puts("[#{__MODULE__}] #{unquote(message)}")                      
    end                                                                   
  end                                                                     
                                                                          
  # Guard definition                                                      
  defguard is_positive(value) when is_integer(value) and value > 0        
  defguardp is_even(value) when is_integer(value) and rem(value, 2) == 0  
                                                                          
  # Function delegation                                                   
  defdelegate parse_int(string), to: String, as: :to_integer              
                                                                          
  # Overridable function                                                  
  defoverridable [process: 1]                                             
                                                                          
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
end   