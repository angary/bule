module AST = struct module T = struct
type cname = string
type vname = string

type eoperator = Add | Div | Log | Max | Min | Mod | Mult | Pow | Sub
type comparison_operator = Lt | Gt | Leq | Geq | Eq | Neq

type expr = VarE of vname | Int of int | BinE of (expr * eoperator * expr)
type term = Exp of expr | Fun of (cname * term list)
type atom = cname * term list
type tuple = ExpTu of expr | FunTu of (cname * tuple list) | Range of (expr * expr)
type atomd = cname * tuple list
type ground_literal = In of atom | Notin of atom | Comparison of (term * comparison_operator * term) | Set of (vname * term)

type glits = ground_literal list

type literal = bool * atom
type literals = glits * bool * atom
type ground_decl = atomd list
type search_decl = bool * expr * atomd list
type clause_decl = literals list * literals list
type hide_decl = bool * literal list

type file =
  { ground: (glits * ground_decl) list;
    prefix: (glits * search_decl) list;
    matrix: (glits * clause_decl) list;
    hide: (glits * hide_decl) list }
end end

module PARSE = struct module T = struct
type ground_literal = In of AST.T.atom | Notin of AST.T.atom | Chain of (AST.T.term * (AST.T.comparison_operator * AST.T.term) list) | Set of (AST.T.vname * AST.T.term)

type glits = ground_literal list
type literals = glits * bool * AST.T.atom
type clause_decl = literals list * literals list

type decl = G of AST.T.ground_decl | S of AST.T.search_decl | C of clause_decl | H of AST.T.hide_decl
type file = (glits * decl) list
end end

module CIRCUIT = struct module T = struct
type ground_term = Fun of (AST.T.cname * ground_term list)
type search_var = AST.T.cname * ground_term list
type quantifier_block = bool * search_var list
type literal = bool * search_var
type clause = literal list * literal list
type file =
  { prefix: quantifier_block list;
    matrix: clause list;
    hide: literal list;
    show: literal list }
end end

module DIMACS = struct module T = struct
module IMap = Map.Make (Int)
module ISet = Set.Make (Int)
module VMap = Map.Make (struct type t = CIRCUIT.T.search_var let compare = compare end)
type search_var = int
type quantifier_block = bool * search_var list
type literal = bool * search_var
type clause = literal list
type file = int * int * quantifier_block list * clause list
end end

