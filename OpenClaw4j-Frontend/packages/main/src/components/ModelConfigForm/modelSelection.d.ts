export interface IModelSelectionValue {
  provider?: string;
  model_id?: string;
}

export interface IModelSelectionOption<T extends IModelSelectionValue> {
  value: string;
  extra: T;
}

export interface IModelSelectionGroup<T extends IModelSelectionValue> {
  options: Array<IModelSelectionOption<T>>;
}

export function findModelOptionByValue<T extends IModelSelectionValue>(
  modelOptions: Array<IModelSelectionGroup<T>>,
  value: IModelSelectionValue,
): IModelSelectionOption<T> | undefined;
