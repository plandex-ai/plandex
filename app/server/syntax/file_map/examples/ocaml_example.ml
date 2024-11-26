(* Module signature *)
module type DataProcessor = sig
  type 'a t
  val create : unit -> 'a t
  val process : 'a t -> 'a -> ('a, string) result
  val validate : 'a -> bool
end

(* Module implementation *)
module StringProcessor : DataProcessor with type 'a = string = struct
type 'a = string
  type t = {
    mutable processed_count: int;
    created_at: float;
  }

  let create () = {
    processed_count = 0;
    created_at = Unix.time ();
  }

  let process t input =
    t.processed_count <- t.processed_count + 1;
    if String.length input > 0 then
      Ok (String.uppercase_ascii input)
    else
      Error "Empty input"

  let validate input =
    String.length input > 0
end

(* Type definitions *)
type user = {
  id: int;
  name: string;
  email: string;
  created_at: float;
}

type 'a result = 
  | Success of 'a 
  | Error of string

(* Variant type *)
type message =
  | Text of string
  | Number of int
  | Tuple of string * int
  | Record of user

(* Functor definition *)
module type Comparable = sig
  type t
  val compare : t -> t -> int
end

module MakeSet (Item : Comparable) = struct
  type element = Item.t
  type t = element list

  let empty = []
  
  let rec add x = function
    | [] -> [x]
    | hd :: tl as l ->
        match Item.compare x hd with
        | 0 -> l
        | n when n < 0 -> x :: l
        | _ -> hd :: add x tl

  let member x set =
    List.exists (fun y -> Item.compare x y = 0) set
end

(* Exception definition *)
exception ValidationError of string

(* Higher-order function *)
let memoize f =
  let cache = Hashtbl.create 16 in
  fun x ->
    try Hashtbl.find cache x
    with Not_found ->
      let result = f x in
      Hashtbl.add cache x result;
      result

(* Object-oriented features *)
class virtual ['a] queue = object(self)
  val mutable items = []
  
  method virtual push : 'a -> unit
  method virtual pop : 'a option
  
  method size = List.length items
  
  method is_empty = items = []
  
  method protected get_items = items
  method protected set_items new_items = items <- new_items
end

class ['a] fifo_queue = object(self)
  inherit ['a] queue

  method push item =
    self#set_items (self#get_items @ [item])

  method pop =
    match self#get_items with
    | [] -> None
    | hd::tl ->
        self#set_items tl;
        Some hd
end

(* Module for handling JSON-like data *)
module Json = struct
  type t =
    | Null
    | Bool of bool
    | Number of float
    | String of string
    | Array of t list
    | Object of (string * t) list

  let rec to_string = function
    | Null -> "null"
    | Bool b -> string_of_bool b
    | Number n -> string_of_float n
    | String s -> "\"" ^ String.escaped s ^ "\""
    | Array items ->
        "[" ^ String.concat ", " (List.map to_string items) ^ "]"
    | Object pairs ->
        let pair_to_string (k, v) =
          "\"" ^ String.escaped k ^ "\": " ^ to_string v
        in
        "{" ^ String.concat ", " (List.map pair_to_string pairs) ^ "}"
end

(* Main execution *)
let () =
  let processor = StringProcessor.create () in
  let result = StringProcessor.process processor "hello world" in
  match result with
  | Ok processed -> Printf.printf "Processed: %s\n" processed
  | Error msg -> Printf.eprintf "Error: %s\n" msg;

  let queue = new fifo_queue in
  queue#push 1;
  queue#push 2;
  queue#push 3;
  
  let rec print_queue () =
    match queue#pop with
    | Some item -> 
        Printf.printf "Item: %d\n" item;
        print_queue ()
    | None -> ()
  in
  print_queue ()


