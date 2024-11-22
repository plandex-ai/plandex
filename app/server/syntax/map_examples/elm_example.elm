module Example exposing (..)

import Browser
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events exposing (..)
import Http
import Json.Decode as Decode exposing (Decoder)
import Json.Encode as Encode

-- MODEL

type alias Model =
    { users : List User
    , inputText : String
    , error : Maybe String
    }

type alias User =
    { id : Int
    , name : String
    , email : String
    }

init : () -> (Model, Cmd Msg)
init _ =
    ( { users = []
      , inputText = ""
      , error = Nothing
      }
    , fetchUsers
    )

-- UPDATE

type Msg
    = GotUsers (Result Http.Error (List User))
    | InputChanged String
    | AddUser
    | UserAdded (Result Http.Error User)

update : Msg -> Model -> (Model, Cmd Msg)
update msg model =
    case msg of
        GotUsers result ->
            case result of
                Ok users ->
                    ( { model | users = users, error = Nothing }
                    , Cmd.none
                    )
                Err _ ->
                    ( { model | error = Just "Failed to fetch users" }
                    , Cmd.none
                    )

        InputChanged text ->
            ( { model | inputText = text }
            , Cmd.none
            )

        AddUser ->
            ( model
            , addUser model.inputText
            )

        UserAdded result ->
            case result of
                Ok user ->
                    ( { model 
                      | users = user :: model.users
                      , inputText = ""
                      , error = Nothing
                      }
                    , Cmd.none
                    )
                Err _ ->
                    ( { model | error = Just "Failed to add user" }
                    , Cmd.none
                    )

-- HTTP

fetchUsers : Cmd Msg
fetchUsers =
    Http.get
        { url = "/api/users"
        , expect = Http.expectJson GotUsers usersDecoder
        }

addUser : String -> Cmd Msg
addUser name =
    Http.post
        { url = "/api/users"
        , body = Http.jsonBody (userEncoder name)
        , expect = Http.expectJson UserAdded userDecoder
        }

-- JSON

userEncoder : String -> Encode.Value
userEncoder name =
    Encode.object
        [ ("name", Encode.string name)
        ]

userDecoder : Decoder User
userDecoder =
    Decode.map3 User
        (Decode.field "id" Decode.int)
        (Decode.field "name" Decode.string)
        (Decode.field "email" Decode.string)

usersDecoder : Decoder (List User)
usersDecoder =
    Decode.list userDecoder

-- VIEW

view : Model -> Html Msg
view model =
    div []
        [ h1 [] [ text "User Management" ]
        , viewError model.error
        , viewInput model.inputText
        , viewUsers model.users
        ]

viewError : Maybe String -> Html Msg
viewError maybeError =
    case maybeError of
        Just error ->
            div [ class "error" ] [ text error ]
        Nothing ->
            text ""

viewInput : String -> Html Msg
viewInput inputText =
    div []
        [ input
            [ value inputText
            , onInput InputChanged
            , placeholder "Enter user name"
            ] []
        , button [ onClick AddUser ] [ text "Add User" ]
        ]

viewUsers : List User -> Html Msg
viewUsers users =
    div []
        [ h2 [] [ text "Users" ]
        , ul [] (List.map viewUser users)
        ]

viewUser : User -> Html Msg
viewUser user =
    li []
        [ text (user.name ++ " (" ++ user.email ++ ")")
        ]

-- MAIN

main : Program () Model Msg
main =
    Browser.element
        { init = init
        , update = update
        , view = view
        , subscriptions = \_ -> Sub.none
        }
